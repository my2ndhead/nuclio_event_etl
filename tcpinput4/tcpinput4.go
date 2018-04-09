package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"
)

// StreamRecord Struct
type StreamRecord struct {
	StreamName string   `json:"StreamName"`
	Records    []Record `json:"Records"`
}

// Record Record Struct
type Record struct {
	ClientInfo   string `json:"ClientInfo"`
	Data         string `json:"Data"`
	PartitionKey string `json:"PartitionKey"`
	ShardID      int    `json:"ShardId"`
}

// handleConnection
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

	var event string

	var streamRecord = StreamRecord{StreamName: "rawevents"}

	var record = make([]Record, 1)

	streamRecord.Records = record

	// Loop over Lines
	for scanner.Scan() {

		// Error handling
		if err := scanner.Err(); err != nil {
			println("Error:", err)
		}
		event = scanner.Text()
		if event != "" {

			url := "http://10.90.1.171:8081/splunk/streams/"

			record[0].Data = base64.StdEncoding.EncodeToString([]byte(event))

			streamRecordJSON, _ := json.Marshal(streamRecord)

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(streamRecordJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-v3io-function", "PutRecords")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			fmt.Println("response Status:", resp.Status)
			fmt.Println("response Headers:", resp.Header)
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println("response Body:", string(body))
		}

		// Reset timeout before looping
		conn.SetReadDeadline(time.Now().Add(timeoutDuration))

	}
	// After the timeout, we should also persist

	print("\nLast Event:\n", event)
}

func doRegexMatch(r *regexp.Regexp, str string) map[string]string {

	match := r.FindStringSubmatch(str)

	if match != nil {
		subMatchMap := make(map[string]string)
		for i, name := range r.SubexpNames() {
			if i != 0 {
				subMatchMap[name] = match[i]
			}
		}
		return subMatchMap

	}
	return nil
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
		port = "12000"
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
