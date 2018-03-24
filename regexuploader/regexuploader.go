package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

type PutItem struct {
	Item Item `json:"Item"`
}

type Item struct {
	Class Class `json:"class"`
	Regex Regex `json:"regex"`
}

type Class struct {
	S string `json:"S"`
}

type Regex struct {
	S string `json:"S"`
}

func main() {
	var putItem PutItem
	var item Item

	file, err := os.Open("/tmp/regexes_ciscoasa.txt")

	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {

		line := scanner.Text()

		// Compiling regex
		_, err := regexp.Compile(line)

		// Catching regex errors
		if err != nil {
			log.Fatal("Regex Error:", line)
		}

		var class Class
		class.S = strconv.Itoa(i)

		var regex Regex
		regex.S = line

		item.Class = class
		item.Regex = regex
		putItem.Item = item

		jsonoutput, _ := json.Marshal(putItem)

		fmt.Println(string(jsonoutput))

		url := `http://10.90.1.171:8081/splunk/conf/props/cisco:asa/extract/` + strconv.Itoa(i)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonoutput))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-v3io-function", "PutItem")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		fmt.Println("HEC", "response Status:: %s", resp.Status)

		bodyHEC, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("HEC", "response Headers:: %s", bodyHEC)

		i = i + 1
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}
