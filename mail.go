package main

import (
	"fmt"
	"io"
	"log"
	"strings"
)

type Header struct {
	Key   string
	Value string
}

func (v Header) String() string {
	return fmt.Sprintf("%s: %s", v.Key, v.Value)
}

type Mail struct {
	Client         string
	From           string
	Recipient      []string
	Headers        []Header
	Body           []string
	HeaderBytes    int64
	BodyBytes      int64
	FromBytes      int64
	RecipientBytes int64
	Timestamp      int64
}

func (v *Mail) WriteTo(dest io.Writer) {
	dest.Write([]byte(fmt.Sprintf("X-SMTP-CLIENT-ADDRESS: %s%s", v.Client, CRLF)))
	dest.Write([]byte(fmt.Sprintf("X-SMTP-RECEVED-AT: %d%s", v.Timestamp, CRLF)))
	dest.Write([]byte(fmt.Sprintf("X-SMTP-ESTIMATED-HEADER-SIZE: %d%s", v.HeaderBytes, CRLF)))
	dest.Write([]byte(fmt.Sprintf("X-SMTP-ESTIMATED-RCPT-SIZE: %d%s", v.RecipientBytes, CRLF)))
	dest.Write([]byte(fmt.Sprintf("X-SMTP-ESTIMATED-BODY-SIZE: %d%s", v.BodyBytes, CRLF)))
	dest.Write([]byte(fmt.Sprintf("X-SMTP-ORGINAL-MAIL-FROM: %s%s", v.From, CRLF)))
	numRecipients := len(v.Recipient)
	for idx, h := range v.Recipient {
		dest.Write([]byte(fmt.Sprintf("X-SMTP-ORGINAL-RCPT-TO-%d-of-%d: %s%s", idx+1, numRecipients, h, CRLF)))
	}
	for _, h := range v.Headers {
		check(dest.Write([]byte(h.String())))
		dest.Write([]byte(CRLF))
	}
	dest.Write([]byte(CRLF))

	for _, b := range v.Body {
		dest.Write([]byte(b))
		dest.Write([]byte(CRLF))
	}
}

func (v *Mail) SetTimeStamp(when int64) {
	v.Timestamp = when
}
func (v *Mail) SetClient(what string) {
	v.Client = what
}

func (v *Mail) SetFrom(where string) {
	v.From = where
	v.FromBytes = int64(len(where))
}

func (v *Mail) AddRecipient(what string) {
	for _, value := range v.Recipient {
		if what == value {
			return
		}
	}
	v.Recipient = append(v.Recipient, what)
	v.RecipientBytes = v.RecipientBytes + int64(len(what))
}

func (v *Mail) AppendHeader(raw string) {
	index := strings.IndexRune(raw, ':')

	if index != -1 {
		key, value := raw[:index], raw[index+1:]
		v.Headers = append(v.Headers, Header{Key: key, Value: value})

	} else {
		if strings.HasPrefix(raw, " ") {
			raw = raw[1:]
		} else {
			v.AppendHeader("invalid-header: " + raw)
		}
		// continuation of previous header!
		if len(v.Headers) > 0 {
			previous := v.Headers[len(v.Headers)-1]
			previous.Value = previous.Value + CRLF + " " + raw
			v.Headers[len(v.Headers)-1] = previous
		} else {
			v.AppendHeader("invalid-header: " + raw)
			return
		}
	}
	v.HeaderBytes += int64(len(raw))
}

func (v *Mail) AppendBody(value string) {
	v.Body = append(v.Body, value)
	v.BodyBytes += int64(len(value))
}

func check(n int, err error) {
	if err != nil {
		log.Printf("Failed to write file %s (written: %d)", err, n)
	}
}

func NewMail(clientAddress string) *Mail {
	return &Mail{From: "", Recipient: make([]string, 0), Headers: make([]Header, 0), Body: make([]string, 0), Client: clientAddress}
}
