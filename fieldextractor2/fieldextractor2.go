package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/nuclio/nuclio-sdk-go"
	"github.com/nuclio/nuclio-test-go"
	"github.com/v3io/v3io-go-http"
)

//********************************

// HECConnection Struct
type HECConnection struct {
	URL            string `json:"url"`
	Authentication string `json:"authentication"`
}

// RegexExtract Struct
type RegexExtract struct {
	Sourcetype string `json:"sourcetype"`
	Class      string `json:"class"`
	Regex      string `json:"regex"`
}

// LogEvent Struct
type LogEvent struct {
	Time       string            `json:"time"`
	Meta       string            `json:"meta"`
	Host       string            `json:"host"`
	Sourcetype string            `json:"sourcetype"`
	Source     string            `json:"source"`
	Index      string            `json:"index"`
	Event      string            `json:"event"`
	Fields     map[string]string `json:"fields"`
}

// ListBucketResult Struct
type ListBucketResult struct {
	Name           string         `xml:"Name"`
	Prefix         string         `xml:"Prefix"`
	Marker         string         `xml:"Marker"`
	Delimiter      string         `xml:"Delimiter"`
	NextMarker     string         `xml:"NextMarker"`
	MaxKeys        string         `xml:"NextMaxKeys"`
	IsTruncated    string         `xml:"IsTruncated"`
	CommonPrefixes []CommonPrefix `xml:"CommonPrefixes"`
}

// CommonPrefix Struct
type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

var regexExtracts []RegexExtract

var myHECConnection HECConnection

// InitContext for setting up function
func InitContext(context *nuclio.Context) error {
	context.UserData = fmt.Sprintf("User data initialized from context: %d", context.WorkerID)

	container := context.DataBinding["db0"].(*v3io.Container)

	// Get Regex Extracts for sourceype
	regexExtracts = getRegexExtracts(container, context)

	myHECConnection = getHTTPEventCollectorConnection(container, context)

	context.Logger.Debug("myHECConnection.URL:", myHECConnection.URL)

	return nil
}

// Handler for HTTP Triggers
func Handler(context *nuclio.Context, event nuclio.Event) (interface{}, error) {

	// Uncomment for testmode
	InitContext(context)

	//container := context.DataBinding["db0"].(*v3io.Container)

	// Get Nuclio Event body
	body := string(event.GetBody())

	// Get Splunk Event Optimizer setting from header (normal, minimal, kv)
	var eventOutputMode = "normal"

	//context.Logger.Debug("Unmarshall LogEvent:", eventOutputMode)

	if event.GetHeader("Event-Output-Mode") != nil {
		// Header types differ between nuclio and nuclio-test invocations
		if _, ok := event.GetHeader("Event-Output-Mode").([]byte); ok {
			eventOutputMode = string(event.GetHeader("Event-Output-Mode").([]byte))
		} else if _, ok := event.GetHeader("Event-Output-Mode").([]uint8); ok {
			eventOutputMode = string(event.GetHeader("Event-Output-Mode").([]uint8))
		} else if _, ok := event.GetHeader("Event-Output-Mode").(string); ok {
			eventOutputMode = event.GetHeader("Event-Output-Mode").(string)
		}
	}

	// Check for empty body
	if len(body) == 0 {
		context.Logger.Debug("Body empty")
		return nuclio.Response{
			StatusCode:  204,
			ContentType: "application/text",

			Body: []byte("Body empty"),
		}, nil
	}

	var logEvent LogEvent

	// Unmarshalling LogEvent

	err := json.Unmarshal([]byte(body), &logEvent)

	// Catching LogEvent unmarshalling errors
	if err != nil {
		context.Logger.Debug("Unmarshall LogEvent:", err)
	}

	// Setting up field key/value map
	logEvent.Fields = map[string]string{}

	// Fetching fields from event
	logEvent = getEventFields(regexExtracts, logEvent, eventOutputMode, context)

	// Fetching internal fields from meta element
	metaFields := getMetaFields(logEvent, context)

	// Adding subsecond resolution to time element
	metaFields.Time = metaFields.Time + metaFields.Fields["_subsecond"]
	fieldsJSON, _ := json.Marshal(metaFields)

	// Throw away non HEC conform data
	metaRegex := `("meta.*?",|"_subsecond.*?",)`
	r, err := regexp.Compile(metaRegex)
	fieldsJSON = []byte(r.ReplaceAllString(string(fieldsJSON), ""))

	context.Logger.Debug("fieldsJSON: %s", fieldsJSON)

	req, err := http.NewRequest("POST", myHECConnection.URL, bytes.NewBuffer(fieldsJSON))
	req.Header.Set("Authorization", myHECConnection.Authentication)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	context.Logger.InfoWith("HEC", "response Status:: %s", resp.Status)
	//context.Logger.InfoWith("HEC", "response Headers:: %s", resp.Header)

	bodyHEC, _ := ioutil.ReadAll(resp.Body)
	context.Logger.InfoWith("HEC", "response Headers:: %s", bodyHEC)

	return nuclio.Response{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        fieldsJSON,
	}, nil

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

func getRegexExtracts(container *v3io.Container, context *nuclio.Context) []RegexExtract {

	listBucketResponse, listBucketerr := container.Sync.ListBucket(&v3io.ListBucketInput{
		Path: "/conf/props/",
	})

	//context.Logger.DebugWith("ListBucketResponse ", "resp", listBucketResponse)
	if listBucketerr != nil {
		context.Logger.ErrorWith("ListBucketerr ", "resp", listBucketerr)
	}

	respBody := listBucketResponse.Body()

	//context.Logger.DebugWith("ListBucketResponse", "respBody", respBody)

	var listBucketResult ListBucketResult

	err := xml.Unmarshal((respBody), &listBucketResult)
	if err != nil {
		context.Logger.ErrorWith("Unmarshal error: %v", err)
	}

	sourcetypeRegex := `props\/(?P<sourcetype>.*?)\/`
	r, _ := regexp.Compile(sourcetypeRegex)

	// Set loop variable to false first
	var last = false

	// Set marker initially to empty
	var marker string

	// Define slice for regexExtracts

	var regexExtracts = make([]RegexExtract, 0)

	// Loop over Regex Classes

	for prefix := range listBucketResult.CommonPrefixes {
		//context.Logger.DebugWith("ListBucketResult", "prefix:", prefix)

		str := listBucketResult.CommonPrefixes[prefix].Prefix

		match := r.FindStringSubmatch(str)

		for i, sourcetype := range match {
			if i != 0 {
				//context.Logger.DebugWith("ListBucketResult", "sourcetype:", sourcetype)

				for last == false {

					GetItemsResponse, GetItemserr := container.Sync.GetItems(&v3io.GetItemsInput{
						Path:           "conf/props/" + sourcetype + "/extract/",
						AttributeNames: []string{"*"},
						Limit:          1000,
						Marker:         marker})

					// Return nil if no regex found for sourcetype
					if GetItemserr != nil {
						context.Logger.WarnWith("Get Item *err*", "err", GetItemserr)
						return nil
					}

					//GetItemsOutput := GetItemsResponse.Output.(*v3io.GetItemsOutput)
					//context.Logger.DebugWith("GetItems ", "resp", GetItemsOutput)

					items := GetItemsResponse.Output.(*v3io.GetItemsOutput).Items

					for item := range items {

						class := items[item]["class"]
						//context.Logger.DebugWith("items", "class", class)

						regex := items[item]["regex"]
						//context.Logger.DebugWith("items", "regex", regex)

						regexExtracts = append(regexExtracts, RegexExtract{sourcetype, class.(string), regex.(string)})

					}

					marker = GetItemsResponse.Output.(*v3io.GetItemsOutput).NextMarker
					last = GetItemsResponse.Output.(*v3io.GetItemsOutput).Last

				}

			}
		}
	}

	return regexExtracts

}

func getHTTPEventCollectorConnection(container *v3io.Container, context *nuclio.Context) HECConnection {
	GetItemResponse, GetItemerr := container.Sync.GetItem(&v3io.GetItemInput{
		Path:           "/conf/outputs/hec/0",
		AttributeNames: []string{"*"}})
	if GetItemerr != nil {
		context.Logger.ErrorWith("Get HEC Connection *err*", "err", GetItemerr)
	} else {
		GetItemOutput := GetItemResponse.Output.(*v3io.GetItemOutput)
		context.Logger.InfoWith("Get HEC Connection ", "resp", GetItemOutput)
	}

	item := GetItemResponse.Output.(*v3io.GetItemOutput).Item

	var myHECConnection HECConnection

	myHECConnection.URL = item["url"].(string)
	myHECConnection.Authentication = item["authorization"].(string)

	return myHECConnection
}

// Function to add meta fields to field list
func getMetaFields(logEvent LogEvent, context *nuclio.Context) LogEvent {

	var fields map[string]string

	// Define regexes for internal fields
	regexExtracts := make([]RegexExtract, 0)
	regexExtracts = append(regexExtracts, RegexExtract{Class: "_subsecond", Regex: "_subsecond::(?P<_subsecond>\\S+)"})
	regexExtracts = append(regexExtracts, RegexExtract{Class: "date_second", Regex: "date_second::(?P<date_second>\\d+)"})
	regexExtracts = append(regexExtracts, RegexExtract{Class: "date_hour", Regex: "date_hour::(?P<date_hour>\\d+)"})
	regexExtracts = append(regexExtracts, RegexExtract{Class: "date_year", Regex: "date_year::(?P<date_second>\\d+)"})
	regexExtracts = append(regexExtracts, RegexExtract{Class: "date_month", Regex: "date_month::(?P<date_month>\\w+)"})
	regexExtracts = append(regexExtracts, RegexExtract{Class: "date_wday", Regex: "date_wday::(?P<date_wday>\\w+)"})
	regexExtracts = append(regexExtracts, RegexExtract{Class: "date_zone", Regex: "date_zone::(?P<date_zone>\\w+)"})

	for _, regexExtract := range regexExtracts {
		//context.Logger.Debug("Meta Regex Extract Name: %v", regexExtract.Class)
		//context.Logger.Debug("Meta Regex Extract Regex: %v", regexExtract.Regex)

		// Compiling regex
		r, err := regexp.Compile(regexExtract.Regex)

		// Catching regex errors
		if err != nil {
			context.Logger.Error("Regex Error:", regexExtract.Regex)
		}

		fields = doRegexMatch(r, logEvent.Meta)
		//context.Logger.Debug("Fields: %s", fields)

		if fields != nil {
			for key, value := range fields {
				logEvent.Fields[key] = value
			}

			//context.Logger.Debug("logEvent: %s", logEvent)
		}
	}
	return logEvent

}

// Function to add event fields to field list
func getEventFields(regexExtracts []RegexExtract, logEvent LogEvent, eventOutputMode string, context *nuclio.Context) LogEvent {

	regexFoundFlag := false

	for i := range regexExtracts {
		if regexExtracts[i].Sourcetype == logEvent.Sourcetype {
			regexFoundFlag = true
		}
	}

	// Nothing to do if regex is not found for event
	if regexFoundFlag == false {
		return logEvent
	}

	var fields map[string]string

	for _, regexExtract := range regexExtracts {

		//context.Logger.Debug("Event Regex Extract Name: %v", regexExtract.Class)
		//context.Logger.Debug("Event Regex Extract Regex: %v", regexExtract.Regex)

		// Compiling regex
		r, err := regexp.Compile(regexExtract.Regex)

		// Catching regex errors
		if err != nil {
			context.Logger.Error("Regex Error:", regexExtract.Regex)
		}

		// Running Regex over
		fields = doRegexMatch(r, logEvent.Event)
		//context.Logger.Debug("Fields: %s", fields)

		if fields != nil {
			for key, value := range fields {
				logEvent.Fields[key] = value
			}

			//context.Logger.Debug("logEvent: %s", logEvent)

		}

	}

	eventOutputMode = "none"

	// Output only segments, drop segmenter characters

	if eventOutputMode == "minimal" {
		logEvent.Event = ""
		for _, value := range logEvent.Fields {
			logEvent.Event = value + " " + logEvent.Event
		}

		segmentersRegex := `[^A-Za-z0-9]`

		r, err := regexp.Compile(segmentersRegex)

		// Catchin regex errors
		if err != nil {
			context.Logger.Error("Regex Error:", segmentersRegex)
		}

		logEvent.Event = r.ReplaceAllString(logEvent.Event, " ")
	} else if eventOutputMode == "kv" {
		logEvent.Event = ""
		for key, value := range logEvent.Fields {
			logEvent.Event = key + "=\"" + value + "\" " + logEvent.Event

		}
	} else if eventOutputMode == "none" {
		logEvent.Event = "-"
	}

	return logEvent
}

func main() {

	data := nutest.DataBind{Name: "db0", Url: "10.90.1.171:8081", Container: "splunk"}

	// Create TestContext and specify the function name, verbose, data
	tc, err := nutest.NewTestContext(Handler, true, &data)
	if err != nil {
		panic(err)
	}

	// Create a new test event
	testEvent := nutest.TestEvent{
		Path:    "/",
		Headers: map[string]interface{}{"Event-Output-Mode": "none"},
		Body: []byte(`
		{
			"time": "1521751024.814",
			"sourcetype": "cisco:asa",
			"meta": "date_second::59 date_hour::22 date_minute::0 date_year::2018 date_month::march date_mday::21 date_wday::wednesday date_zone::60 mytestfield1::bla mytestfield1::\"bla\"", 
			"host": "myhost",
			"source": "127.0.0.1",
			"index": "main",
			"event": "Mar 23 19:59:58 pix-inside %PIX-4-106023: Deny protocol 4 src outside:210.217.159.25 dst inside :10.87.80.86 by access-group \"ACL-FROM-OUTSIDE\""
		}`),
	}

	// Invoke the tested function with the new event and log it's output
	resp, err := tc.Invoke(&testEvent)

	// Get body as string
	responseBody := string(resp.(nuclio.Response).Body)

	// Log results
	tc.Logger.InfoWith("Run complete", "Body", responseBody, "err", err)
}
