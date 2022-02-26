# gosmtp
SMTP server written in GO. It doesn't send mail, it acts like a SMTP, so your program can test integration with SMTP servers.

It stores mails in the "mails" folder by default for viewing.

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

## Running only plaintext (port 25)
```bash
# $GOPATH/bin/gosmtp
```

## Running only tls (port 465)
```bash
# $GOPATH/bin/gosmtp --tls-cert=cert.pem --tls-key=cert.key --port=-1
```

# Getting certs
The default certs `cert.pem`, `cert.key` is only valid for `127.0.0.1` or `localhost`, they are valid for 20 years from 2022

To get proper CA issued cert, please use let's encrypt (https://www.letsencrypt.org)
To generate self-signed CA & certs, You can use openssl to generate certs your self.

We recommend you using minica, a free open source graphical CA for the techies. Visit https://github.com/wushilin/minica



