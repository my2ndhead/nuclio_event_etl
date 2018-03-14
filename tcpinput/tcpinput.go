package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"time"
)

func handleConnection(conn net.Conn) {
	fmt.Println("Handling new connection...")

	fmt.Println(conn.RemoteAddr())

	// Close connection when this function ends
	defer func() {
		fmt.Println("Closing connection...")
		conn.Close()
	}()

	// Set timeout to 5 seconds
	timeoutDuration := 30 * time.Second

	// Create new buffered reader
	bufReader := bufio.NewReader(conn)

	// Create linescanner
	scanner := bufio.NewScanner(bufReader)

	// Regex do find beginning of new event, usefull for multiline events
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

	var event string

	f, _ := os.Create("/tmp/event")
	defer f.Close()

	// Loop over Lines
	for scanner.Scan() {

		// Search for new events
		match := re.FindStringSubmatch(scanner.Text())
		if match != nil { // New event starts
			if event != "" {
				// If we have already collected an event, we should persist it somwhere
				print("\nEvent:\n", event)
				f.WriteString(event)
				f.Sync()
			}
			// Store first line of new event
			event = scanner.Text()
		} else {
			// Append next line to event
			event = event + "\n" + scanner.Text()
		}

		// Error handling
		if err := scanner.Err(); err != nil {
			println("Error:", err)
		}

		// Reset timeout before looping
		conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	}
	// After the timeout, we should also persist
	print("\nEvent:\n", event)
	f.WriteString(event)
	f.Sync()

}

func main() {

	// Make Bindadress configurable
	bindAddr := os.Getenv("TCPINPUT_BINDADDR")

	// Make Port configurable
	port := os.Getenv("TCPINPUT_PORT")

	// Define default Bindadress
	if bindAddr == "" {
		bindAddr = "0.0.0.0"
	}

	//Define default port
	if port == "" {
		port = "8888"
	}

	// Create listener
	listener, err := net.Listen("tcp", bindAddr+":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		listener.Close()
		fmt.Println("Listener closed")
	}()

	for {
		// Get net.TCPConn object
		conn, err := listener.Accept()

		if err != nil {
			fmt.Println(err)
			break
		}

		// Run the connection handler
		go handleConnection(conn)
	}

}
