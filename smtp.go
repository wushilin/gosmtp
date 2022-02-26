package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var sequence int64 = 0
var port_number = flag.Int("port", 25, "The smtp server port")
var secure_port_number = flag.Int("secure-port", 465, "The smtp secure port with tls")
var save_dir = flag.String("save-to", "mails", "The directory to save mails to")
var bind_address = flag.String("bind", "", "The bind address. Defaults to all interface")
var cert = flag.String("tls-cert", "", "The TLS cert")
var key = flag.String("tls-key", "", "The TLS Key")
var max_body_size = flag.Int64("max-body-size", 50*1024*1024, "The max body size, default 50MiB")
var max_header_size = flag.Int64("max-header-size", 1024*1024, "The max header size, default 1MiB")
var max_recipient_size = flag.Int64("max-recipient-size", 1024*1024, "The max RCPT to size, default 1MiB")
var verbose = flag.Bool("verbose", false, "Show debug or not")

var sigs = make(chan os.Signal, 1)
var stop = false

var active_client_count int64 = 0

func printFlags() {
	fmt.Printf("Config: port                  => %d\n", *port_number)
	fmt.Printf("Config: secure-port           => %d\n", *secure_port_number)
	fmt.Printf("Config: save-to               => %s\n", *save_dir)
	fmt.Printf("Config: bind                  => %s\n", *bind_address)
	fmt.Printf("Config: tls-cert              => %s\n", *cert)
	fmt.Printf("Config: tls-key               => %s\n", *key)
	fmt.Printf("Config: max-body-size         => %d\n", *max_body_size)
	fmt.Printf("Config: max-header-size       => %d\n", *max_header_size)
	fmt.Printf("Config: max-recipient-size    => %d\n", *max_recipient_size)
	fmt.Printf("Config: verbose               => %t\n", *verbose)
}
func main() {
	flag.Parse()
	printFlags()
	if *save_dir == "" {
		log.Fatalf("You must specify a save directory with -saveto xxx")
	}

	CreateDirectoryIfNotThere(*save_dir)
	if strings.HasSuffix(*save_dir, string(os.PathSeparator)) {
		*save_dir = strings.TrimSuffix(*save_dir, string(os.PathSeparator))
	}
	log.Printf("Saving mails to %s", *save_dir)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// This goroutine executes a blocking receive for
		// signals. When it gets one it'll print it out
		// and then notify the program that it can finish.
		sig := <-sigs
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>ALERT<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
		log.Println(sig)
		log.Println("Waiting for graceful shutdown - Press CTRL-C or kill again to force quit")
		stop = true
		signal.Stop(sigs)
	}()

	var wg sync.WaitGroup
	if *port_number != -1 {
		listen_address := fmt.Sprintf("%s:%d", *bind_address, *port_number)

		normal, err := net.Listen("tcp", listen_address)
		if err != nil {
			log.Fatalf("Failed to start normal listener on %s. Error: %s", listen_address, err)
		}
		wg.Add(1)
		go handle(normal, &wg)
	}

	if *secure_port_number != -1 && *cert != "" && *key != "" {
		x509_cert, err := tls.LoadX509KeyPair(*cert, *key)

		if err != nil {
			log.Fatalf("Failed to load cert/key pair. cert: %s key: %s, error: %s", *cert, *key, err)
		}
		config := tls.Config{Certificates: []tls.Certificate{x509_cert}}

		listen_address := fmt.Sprintf("%s:%d", *bind_address, *secure_port_number)

		secure, err := tls.Listen("tcp", listen_address, &config)
		if err != nil {
			log.Fatalf("Failed to start secure listener on %s. Error: %s", listen_address, err)
		}
		wg.Add(1)
		go handle(secure, &wg)
	}

	wg.Wait()
	waitCounter := 0
	for {
		if active_client_count == 0 {
			break
		}
		if waitCounter%50 == 0 {
			log.Printf("%d client(s) active\n", active_client_count)
		}
		time.Sleep(100 * time.Millisecond)
		waitCounter++
	}
	log.Printf("Bye.")
}

func listenWithChannel(listener net.Listener, channel chan net.Conn) {
	for !stop {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		channel <- conn
	}
	close(channel)
}

func handle(listener net.Listener, wg *sync.WaitGroup) {
	log.Printf("Started listener on address %s", listener.Addr())
	defer wg.Done()
	socks := make(chan net.Conn, 50000)
	go listenWithChannel(listener, socks)

	newwg := sync.WaitGroup{}
	for !stop {
		select {
		case conn, ok := <-socks:
			if !ok {
				break
			}
			newwg.Add(1)
			atomic.AddInt64(&active_client_count, 1)
			go handleConn(conn, &newwg)
		case <-time.After(500 * time.Millisecond):
		}
	}
	listener.Close()
	log.Printf("Stopped listener on address %s", listener.Addr())
}

const WELCOME = "220 simple.smtp.server welcomes you" + CRLF

func handleConn(conn net.Conn, wg *sync.WaitGroup) {
	defer func() {
		atomic.AddInt64(&active_client_count, -1)
		conn.Close()
		log.Printf("Done handling connection from %+v", conn.RemoteAddr())
	}()
	log.Printf("Handling connection from %+v", conn.RemoteAddr())

	defer wg.Done()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writer.WriteString(WELCOME)
	writer.Flush()
	buffer := make([]byte, 4096)
	address := fmt.Sprintf("%+v", conn.RemoteAddr())
	for !stop {
		shouldContinue := handleSession(reader, writer, buffer, address)
		if !shouldContinue {
			return
		}
	}
}

func debug(msg string, args ...interface{}) {
	if *verbose {
		log.Printf(msg, args...)
	}
}
func handleSession(reader *bufio.Reader, writer *bufio.Writer, buffer []byte, address string) bool {
	mail := NewMail(address)
	// command mode
	for {
		line, err := readLineFrom(reader, buffer)
		debug("Received %s", line)
		if err != nil {
			break
		}
		if strings.ToUpper(line) == "DATA" {
			// data mode
			reply(writer, "354 End data with <CR><LF>.<CR><LF>")
			break
		}
		command, err := ParseCommand(line)
		if err != nil {
			reply(writer, "501 Sorry, what did you say?")
			continue
		}
		switch command.Verb {
		case "MAIL":
			who := command.Argument
			who = strings.TrimSpace(who)
			index := strings.IndexRune(who, ':')
			from := who
			if index != -1 {
				from = strings.TrimSpace(who[index+1:])
			}
			mail.SetFrom(from)
			debug("From set as %s", from)
		case "RCPT":
			who := command.Argument
			who = strings.TrimSpace(who)
			index := strings.IndexRune(who, ':')
			recipient := who
			if index != -1 {
				recipient = strings.TrimSpace(who[index+1:])
			}
			mail.AddRecipient(recipient)
			if mail.RecipientBytes > *max_recipient_size {
				reply(writer, "521 too many recipients will fail")
				return false
			}
			debug("New recipient added %s", recipient)
		case "RSET":
			mail = NewMail(address)
			debug("Mail discarded")
		case "QUIT":
			debug("Client request disconnect")
			response := process(command)
			reply(writer, response.ToString())
			return false
		}
		response := process(command)
		reply(writer, response.ToString())
	}
	// handle data
	headerMode := true
	for {
		line, err := readLineFrom(reader, buffer)
		if err != nil {
			return false
		}
		if line == "." {
			debug("Client ended mail transaction")
			reply(writer, "250 Swallowed")
			break
		}
		if headerMode {
			if line == "" {
				headerMode = false
			} else {
				mail.AppendHeader(line)
				if mail.HeaderBytes > *max_header_size {
					reply(writer, "521 too many headers will result in too much memory")
					return false
				}
			}
		} else {
			mail.AppendBody(line)
			if mail.BodyBytes > *max_body_size {
				reply(writer, "521 too much data is bad for health")
				return false
			}
		}
	}
	file, written := handleMail(mail)
	log.Printf("Written %d bytes to file %s", written, file)
	return true
}

func handleMail(mail *Mail) (filename string, written int64) {
	now := time.Now()
	mail.SetTimeStamp(now.Unix())
	filename = now.Format("2006-01-02T15:04:05")
	sequence := atomic.AddInt64(&sequence, 1)
	filename = filename + "-" + fmt.Sprintf("%06d", sequence%100000) + ".eml"

	filename = *save_dir + string(os.PathSeparator) + filename
	fh, err := os.Create(filename)
	if err != nil {
		log.Printf("Create file error %s", err)
		return filename, 0
	}
	defer func() {
		fh.Close()
		stat, err := os.Stat(filename)
		if err == nil {
			written = stat.Size()
		}
	}()
	mail.WriteTo(fh)
	return filename, 0
}
func reply(writer *bufio.Writer, what string) {
	debug("Replied: %s", what)
	writer.WriteString(what)
	writer.WriteString(CRLF)
	writer.Flush()
}
func process(cmd Command) Response {
	switch cmd.Verb {
	case "HELO":
		return NewResponse("250", fmt.Sprintf("Hello %s, I am just a dummy SMTP are you sure to proceed?", cmd.Argument))
	case "ELHO":
		return NewResponse("502", "What do you expect?")
	case "MAIL":
		return NewResponse("250", "Hey you, we got you")
	case "RCPT":
		return NewResponse("250", "That name looks familiar")
	case "RSET":
		return NewResponse("250", "All effort was wasted")
	case "VRFY":
		return NewResponse("250", "Does true really mean true and false is really false?")
	case "NOOP":
		return NewResponse("250", "Yawn yawn... so board")
	case "QUIT":
		return NewResponse("221", "Your mail will be swallowed. Grrrr")
	default:
		return NewResponse("502", "Sorry, we don't support that fancy stuff")
	}
}
func readLineFrom(reader *bufio.Reader, buffer []byte) (string, error) {
	readCount := 0
	remaining := len(buffer)
	run := true
	for run {
		bytes, isPrefix, err := reader.ReadLine()
		if err != nil {
			return "", err
		}
		toCopy := min(len(bytes), remaining)
		copyBytes(buffer, readCount, bytes, 0, toCopy)
		readCount += toCopy
		remaining -= toCopy
		if !isPrefix {
			run = false
		}
	}
	return string(buffer[:readCount]), nil
}

func min(a1 int, a2 int) int {
	if a1 < a2 {
		return a1
	}
	return a2
}

func copyBytes(dest []byte, destOffset int, src []byte, srcOffset int, count int) {
	for i := 0; i < count; i++ {
		dest[destOffset+i] = src[srcOffset+i]
	}
}
