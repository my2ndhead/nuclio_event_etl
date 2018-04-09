package main

import nuclio "github.com/nuclio/nuclio-sdk-go"

// Handler for Stream events
func Handler(context *nuclio.Context, event nuclio.Event) (interface{}, error) {
	println(event.GetTotalNumShards())

	return nuclio.Response{}
}

func main() {

}
