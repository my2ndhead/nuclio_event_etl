// @nuclio.configure
//
// function.yaml:
//   apiVersion: "nuclio.io/v1"
//   kind: "Function"
//   spec:
//     runtime: "golang"
//     triggers:
//       http:
//         maxWorkers: 8
//         kind: http
//         attributes:
//           ingresses:
//             first:
//               paths:
//               - /first/path
//               - /second/path
//             second:
//               host: my.host.com
//               paths:
//               - /first/from/host
//		dataBindings:
//		  db0:
//		    class: v3io
//	        url: http://10.90.1.171:8081

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
	Sourcetype string            `json:"sourcetype"`
	Host       string            `json:"host"`
	Source     string            `json:"source"`
	Event      string            `json:"event"`
	Fields     map[string]string `json:"fields"`
}

// Handler for HTTP Triggers
func Handler(context *nuclio.Context, event nuclio.Event) (interface{}, error) {

	container := context.DataBinding["db0"].(*v3io.Container)

	// Set loop variable to false first
	var last = false

	// Set marker initially to empty
	var marker string

	// Define sourcetype (currently static)
	var sourcetype = "mysourcetype"

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

	context.Logger.DebugWith("RegexExtracts", "Array", regexExtracts)

	// Get Nuclio Event body
	body := string(event.GetBody())

	// Check for empty body
	if len(body) == 0 {
		context.Logger.Debug("Body empty")
		return nuclio.Response{
			StatusCode:  204,
			ContentType: "application/text",

			Body: []byte("Body empty"),
		}, nil
	}

	// Unmarshalling LogEvent
	var logEvent LogEvent

	err := json.Unmarshal([]byte(body), &logEvent)

	// Catching LogEvent unmarshalling errors
	if err != nil {
		context.Logger.Debug("Unmarshall LogEvent:", err)
	}

	// Debug logging LogEvent content
	context.Logger.Debug("Body: %s", body)
	context.Logger.Debug("Time: %s", logEvent.Time)
	context.Logger.Debug("Sourcetype: %s", logEvent.Sourcetype)
	context.Logger.Debug("Host: %s", logEvent.Host)
	context.Logger.Debug("Source: %s", logEvent.Source)
	context.Logger.Debug("Event: %s", logEvent.Event)

	// Setting up field key/value map
	logEvent.Fields = map[string]string{}

	// Running regexes over raw event
	context.Logger.Debug("Regex Extract Count: %v", len(regexExtracts))

	for index, regexExtract := range regexExtracts {
		context.Logger.Debug("Index: %v", index)
		context.Logger.Debug("regexExtract: %v", regexExtract)

	}

	var fieldsJSON []byte

	for index, regexExtract := range regexExtracts {
		context.Logger.Debug("Index: %v", index)

		context.Logger.Debug("Regex Extract Name: %v", regexExtract.Class)
		context.Logger.Debug("Regex Extract Regex: %v", regexExtract.Regex)

		// Compuling regex
		r, err := regexp.Compile(regexExtract.Regex)

		// Catchin regex errors
		if err != nil {
			return nuclio.Response{
				StatusCode:  500,
				ContentType: "application/text",
				Body:        []byte("Regex error"),
			}, nil
		}

		// Running Regex over
		fields := reSubMatchMap(r, logEvent.Event)
		context.Logger.Debug("*************Fields: %s", fields)
		//var extractedField LogEventField

		if fields != nil {
			for key, value := range fields {
				context.Logger.Debug("Field: %s Value: %s", key, value)
				logEvent.Fields[key] = value
				context.Logger.Debug("logEvent.Fields: %s", logEvent.Fields)
			}
			context.Logger.Debug("logEvent: %s", logEvent)
			// Format into JSON
			fieldsJSON, _ = json.Marshal(logEvent)
			context.Logger.Debug("fieldsJSON: %s", fieldsJSON)
		}
	}

	if fieldsJSON != nil {
		return nuclio.Response{
			StatusCode:  200,
			ContentType: "application/json",
			Body:        []byte(fieldsJSON),
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

func reSubMatchMap(r *regexp.Regexp, str string) map[string]string {

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

	data := nutest.DataBind{Name: "db0", Url: "10.90.1.171:8081", Container: "splunk"}

	// Create TestContext and specify the function name, verbose, data
	tc, err := nutest.NewTestContext(Handler, true, &data)
	if err != nil {
		panic(err)
	}

	// Create a new test event
	testEvent := nutest.TestEvent{
		Path: "/",
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
