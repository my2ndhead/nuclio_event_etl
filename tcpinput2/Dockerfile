FROM golang:1.9.2
WORKDIR /tmp/tcpinput2
COPY fetch_unconfirmed_transactions.go .
RUN go build -o tcpinput2 tcpinput2.go
CMD ["./tcpinput2"]
