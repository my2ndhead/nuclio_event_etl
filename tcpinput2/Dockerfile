FROM golang:1.9.2
WORKDIR /tmp/tcpinput2
COPY tcpinput2.go .
RUN go build -o tcpinput2 tcpinput2.go
CMD ["./tcpinput2"]
