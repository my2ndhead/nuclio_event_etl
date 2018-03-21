package main

import (
	"encoding/json"
	"regexp"

	"github.com/nuclio/nuclio-sdk-go"
	"github.com/nuclio/nuclio-test-go"
	"github.com/v3io/v3io-go-http"
)

//********************************

// RegexExtract Struct
type RegexExtract struct {
	Class string `json:"class"`
	Regex string `json:"regex"`
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

// Handler for HTTP Triggers
func Handler(context *nuclio.Context, event nuclio.Event) (interface{}, error) {

	container := context.DataBinding["db0"].(*v3io.Container)

	// Get Nuclio Event body
	body := string(event.GetBody())

	// Get Splunk Event Optimizer setting from header
	var optimizeEvent = "false"

	if event.GetHeader("Optimize-Event") != nil {
		optimizeEvent = event.GetHeader("Optimize-Event").(string)
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

	// Debug logging LogEvent content
	/*context.Logger.Debug("Body: %s", body)
	context.Logger.Debug("Time: %s", logEvent.Time)
	context.Logger.Debug("Sourcetype: %s", logEvent.Sourcetype)
	context.Logger.Debug("Host: %s", logEvent.Host)
	context.Logger.Debug("Source: %s", logEvent.Source)
	context.Logger.Debug("Event: %s", logEvent.Event)*/

	// Setting up field key/value map
	logEvent.Fields = map[string]string{}

	// Define sourcetype (currently static)
	sourcetype := logEvent.Sourcetype

	// Get Regex Extracts for sourceype
	var regexExtracts = getRegexExtracts(sourcetype, container, context)

	// Running regexes over raw event

	var eventWithFields = getEventWithFields(regexExtracts, logEvent, optimizeEvent, context)

	if eventWithFields != nil {
		return nuclio.Response{
			StatusCode:  200,
			ContentType: "application/json",
			Body:        []byte(eventWithFields),
		}, nil
	}

	// Catch non matches and return 204
	context.Logger.Debug("Body empty")
	return nuclio.Response{
		StatusCode:  204,
		ContentType: "application/text",
		Body:        []byte("Body empty"),
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

func getRegexExtracts(sourcetype string, container *v3io.Container, context *nuclio.Context) []RegexExtract {

	// Set loop variable to false first
	var last = false

	// Set marker initially to empty
	var marker string

	// Define slice for regexExtracts

	var regexExtracts = make([]RegexExtract, 0)

	// Loop over Regex Classes

	for last == false {

		GetItemsResponse, GetItemserr := container.Sync.GetItems(&v3io.GetItemsInput{
			Path:           "/conf/" + sourcetype + "/extract/",
			AttributeNames: []string{"*"},
			Limit:          1000,
			Marker:         marker})

		if GetItemserr != nil {
			context.Logger.ErrorWith("Get Item *err*", "err", GetItemserr)
		} else {
			GetItemsOutput := GetItemsResponse.Output.(*v3io.GetItemsOutput)
			context.Logger.DebugWith("GetItems ", "resp", GetItemsOutput)
		}

		items := GetItemsResponse.Output.(*v3io.GetItemsOutput).Items

		for item := range items {

			class := items[item]["class"]
			context.Logger.DebugWith("items", "class", class)

			regex := items[item]["regex"]
			context.Logger.DebugWith("items", "regex", regex)

			regexExtracts = append(regexExtracts, RegexExtract{class.(string), regex.(string)})

		}

		marker = GetItemsResponse.Output.(*v3io.GetItemsOutput).NextMarker
		last = GetItemsResponse.Output.(*v3io.GetItemsOutput).Last

	}

	return regexExtracts

}

func getEventWithFields(regexExtracts []RegexExtract, logEvent LogEvent, optimizeEvent string, context *nuclio.Context) []byte {
	var fieldsJSON []byte
	var fields map[string]string

	for _, regexExtract := range regexExtracts {

		context.Logger.Debug("Regex Extract Name: %v", regexExtract.Class)
		context.Logger.Debug("Regex Extract Regex: %v", regexExtract.Regex)

		// Compiling regex
		r, err := regexp.Compile(regexExtract.Regex)

		// Catchin regex errors
		if err != nil {
			context.Logger.Error("Regex Error:", regexExtract.Regex)
			return nil
		}

		// Running Regex over
		fields = doRegexMatch(r, logEvent.Event)
		context.Logger.Debug("Fields: %s", fields)
		//var extractedField LogEventField

		if fields != nil {
			for key, value := range fields {
				logEvent.Fields[key] = value
			}

			context.Logger.Debug("logEvent: %s", logEvent)

		}

	}

	if optimizeEvent == "true" {
		logEvent.Event = ""
		for _, value := range logEvent.Fields {
			logEvent.Event = logEvent.Event + " " + value
		}

		segmentersRegex := `[.,:;]`

		r, err := regexp.Compile(segmentersRegex)

		// Catchin regex errors
		if err != nil {
			context.Logger.Error("Regex Error:", segmentersRegex)
			return nil
		}

		logEvent.Event = r.ReplaceAllString(logEvent.Event, "")
	}

	// Format into JSON
	fieldsJSON, _ = json.Marshal(logEvent)
	context.Logger.Debug("fieldsJSON: %s", fieldsJSON)

	return fieldsJSON

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
		Headers: map[string]interface{}{"Optimize-Event": "true"},
		Body: []byte(`
		{
			"time": "15000000000.500",
			"sourcetype": "mysourcetype",
			"host": "myhost",
			"source": "mysource",
			"event": "name=\"Kent\" firstname=\"Clark\" address=\"101 mainstreet, New York\""
		}`),
	}

	// Invoke the tested function with the new event and log it's output
	resp, err := tc.Invoke(&testEvent)

	// Get body as string
	responseBody := string(resp.(nuclio.Response).Body)

	// Log results
	tc.Logger.InfoWith("Run complete", "Body", responseBody, "err", err)
}
