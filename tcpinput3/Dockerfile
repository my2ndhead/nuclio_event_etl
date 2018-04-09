FROM golang:1.9.2
WORKDIR /tmp/tcpinput3
COPY tcpinput2.go .
RUN go build -o tcpinput3 tcpinput3.go
CMD ["./tcpinput3"]
