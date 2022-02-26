# gosmtp
SMTP server written in GO. It doesn't send mail, but act as a mail server for you to integrate with!

# Installing
```bash
$ go install github.com/wushilin/gosmtp@v1.0.0
```

# Running

Note port 25, 465 requires ROOT!

## Running with both TLS and plaintext(all options)
```bash
# $GOPATH/bin/gosmtp --tls-cert=cert.pem --tls-key=cert.key --port=25 --secure-port=465 --save-to mails --bind "" --max-body-size=100000000 --max-header-size=100000 --max-recipient-size=1000000 --verbose=true
```

## Running only plaintext
```bash
# $GOPATH/bin/gosmtp
```

## Running only tls
```bash
# $GOPATH/bin/gosmtp --tls-cert=cert.pem --tls-key=cert.key --port=-1
```

# Getting certs
You can use openssl to generate certs your self, or use minica: github.com/wushilin/minica


