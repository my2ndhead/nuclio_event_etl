package main

import (
	"encoding/json"
	"regexp"

	"github.com/nuclio/nuclio-sdk-go"
	"github.com/nuclio/nuclio-test-go"
)

// Handler for HTTP Triggers
func Handler(context *nuclio.Context, event nuclio.Event) (interface{}, error) {

	// *****************************************
	// Test JSON
	regexJSON := `
	[
		{ 
		    "name": "name,firstname",
		    "regex": "name=\"(?P<name>\\w+).*?\"\\s+firstname=\"(?P<Firstname>\\w+)\""
		},
		{ 
			"name": "address",
		    "regex": "address=\"(?P<address>.*?)\""
		}
	]
	`
	//********************************

	// Define Structs
	type RegexExtract struct {
		Name  string
		Regex string
	}

	/*type LogEventField struct {
		FieldName  string
		FieldValue string
	}*/

	type LogEvent struct {
		Time       string `json:"time"`
		Sourcetype string `json:"sourcetype"`
		Host       string `json:"host"`
		Source     string `json:"source"`
		Event      string `json:"event"`
		//	Fields     []LogEventField `json:"fields"`
		Fields map[string]string `json:"fields"`
	}

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

	var regexExtracts []RegexExtract

	err := json.Unmarshal([]byte(regexJSON), &regexExtracts)

	if err != nil {
		context.Logger.Debug("Unmarshall regexExtracts:", err)
	}

	for l := range regexExtracts {
		context.Logger.Debug("Regex Extract Name: %v", regexExtracts[l].Name)
		context.Logger.Debug("Regex Extract Regex: %v", regexExtracts[l].Regex)
	}

	var logEvent LogEvent

	err = json.Unmarshal([]byte(body), &logEvent)

	if err != nil {
		context.Logger.Debug("Unmarshall LogEvent:", err)
	}

	context.Logger.Debug("Body: %s", body)
	context.Logger.Debug("Time: %s", logEvent.Time)
	context.Logger.Debug("Sourcetype: %s", logEvent.Sourcetype)
	context.Logger.Debug("Host: %s", logEvent.Host)
	context.Logger.Debug("Source: %s", logEvent.Source)
	context.Logger.Debug("Event: %s", logEvent.Event)

	//logEvent.Fields = make([]LogEventField, 1)

	logEvent.Fields = map[string]string{}

	for l := range regexExtracts {
		context.Logger.Debug("Regex Extract Name: %v", regexExtracts[l].Name)
		context.Logger.Debug("Regex Extract Regex: %v", regexExtracts[l].Regex)

		r, err := regexp.Compile(regexExtracts[l].Regex)

		if err != nil {
			return nuclio.Response{
				StatusCode:  500,
				ContentType: "application/text",
				Body:        []byte("Regex error"),
			}, nil
		}

		fields := reSubMatchMap(r, logEvent.Event)

		//var extractedField LogEventField

		if fields != nil {
			for key, value := range fields {
				context.Logger.Debug("Field: %s Value: %s", key, value)
				logEvent.Fields[key] = value
				context.Logger.Debug("logEvent.Fields: %s", logEvent.Fields)
			}
			context.Logger.Debug("logEvent: %s", logEvent)
			// Format into JSON
			fieldsJSON, _ := json.Marshal(logEvent)
			context.Logger.Debug("fieldsJSON: %s", fieldsJSON)

			return nuclio.Response{
				StatusCode:  200,
				ContentType: "application/json",
				Body:        []byte(fieldsJSON),
			}, nil
		}

	}

	// Catch non matches and return 204
	context.Logger.Debug("Body empty")
	return nuclio.Response{
		StatusCode:  204,
		ContentType: "application/text",
		Body:        []byte("Body empty"),
	}, nil

	return nuclio.Response{
		StatusCode:  200,
		ContentType: "application/text",
		Body:        []byte("Done"),
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
	// Create TestContext and specify the function name, verbose, data
	tc, err := nutest.NewTestContext(Handler, true, nil)
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
