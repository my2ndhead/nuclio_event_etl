package main

import (
	"bufio"
	"bytes"
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

	var fields map[string]string

	regex := `time=(?P<time>.*?)\|meta=(?P<meta>.*?)\|host=(?P<host>.*?)\|sourcetype=(?P<sourcetype>.*?)\|source=(?P<source>.*?)\|index=(?P<index>.*?)\|(?P<event>.*?)###END###$`

	r, _ := regexp.Compile(regex)

	f, _ := os.Create("/tmp/event")
	defer f.Close()

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

			/*if fields != nil {
				for key, value := range fields {
					print("\nKey:\n", key)
					print("\nValue:\n", value)
				}
			}*/

			logEvent.Time = fields["time"]
			logEvent.Meta = fields["meta"]
			logEvent.Host = fields["host"]
			logEvent.Sourcetype = fields["sourcetype"]
			logEvent.Source = fields["source"]
			logEvent.Index = fields["index"]
			logEvent.Event = fields["event"]

			logEventJSON, _ := json.Marshal(logEvent)

			bla := string(logEventJSON)

			fmt.Println("Event:", bla)

			url := "http://fieldextractor2.lcsystems:32327"

			req, err := http.NewRequest("POST", url, bytes.NewBuffer(logEventJSON))
			req.Header.Set("X-Custom-Header", "myvalue")
			req.Header.Set("Content-Type", "application/json")

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

			//print("\nErr:\n", err)

			//f.WriteString(fields)
			//f.Sync()
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

	print("\nEvent:\n", logEventJSON)

	//f.WriteString(event)
	//f.Sync()
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
