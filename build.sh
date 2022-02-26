#!/bin/sh

go build
rm -rf mails
./gosmtp --tls-cert=cert.pem --tls-key=cert.key --port=25 --secure-port=465 --save-to mails --bind "" --max-body-size=100000000 --max-header-size=100000 --max-recipient-size=1000000 --verbose=true
