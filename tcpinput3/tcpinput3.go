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

// LogEvent Struct
type LogEvent struct {
	Time       string `json:"time"`
	Meta       string `json:"meta"`
	Host       string `json:"host"`
	Sourcetype string `json:"sourcetype"`
	Source     string `json:"source"`
	Index      string `json:"index"`
	Event      string `json:"event"`
}

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

	var logEvent LogEvent

	var streamRecord = StreamRecord{StreamName: "eventinput"}

	var record = make([]Record, 1)

	streamRecord.Records = record

	var fields map[string]string

	regex := `time=(?P<time>.*?)\|meta=(?P<meta>.*?)\|host=(?P<host>.*?)\|sourcetype=(?P<sourcetype>.*?)\|source=(?P<source>.*?)\|index=(?P<index>.*?)\|(?P<event>.*?)$`

	r, _ := regexp.Compile(regex)

	// Loop over Lines
	for scanner.Scan() {

		// Error handling
		if err := scanner.Err(); err != nil {
			println("Error:", err)
		}
		event = scanner.Text()
		if event != "" {

			// Running Regex over
			fields = doRegexMatch(r, event)

			logEvent.Time = fields["time"]
			logEvent.Meta = fields["meta"]
			logEvent.Host = fields["host"]
			logEvent.Sourcetype = fields["sourcetype"]
			logEvent.Source = fields["source"]
			logEvent.Index = fields["index"]
			logEvent.Event = fields["event"]

			logEventJSON, _ := json.Marshal(logEvent)

			record[0].Data = base64.StdEncoding.EncodeToString(logEventJSON)

			streamRecordJSON, _ := json.Marshal(streamRecord)

			url := "http://10.90.1.171:8081/splunk/streams/"

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

	logEvent.Time = fields["time"]
	logEvent.Meta = fields["meta"]
	logEvent.Host = fields["host"]
	logEvent.Sourcetype = fields["sourcetype"]
	logEvent.Source = fields["source"]
	logEvent.Index = fields["index"]
	logEvent.Event = fields["event"]

	logEventJSON, _ := json.Marshal(logEvent)

	record[0].Data = base64.StdEncoding.EncodeToString(logEventJSON)

	streamRecordJSON, _ := json.Marshal(streamRecord)

	url := "http://10.90.1.171:8081/splunk/streams/"

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

	print("\nEvent:\n", logEventJSON)
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
