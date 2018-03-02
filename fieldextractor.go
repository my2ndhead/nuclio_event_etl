package nuclio_etl

import (
    "encoding/json"
    "regexp"
    "github.com/nuclio/nuclio-sdk-go"
)

func Handler(context *nuclio.Context, event nuclio.Event) (interface{}, error) {

	// Unstructured debug message
	//context.Logger.Debug("Process document %s, length %d", event.GetPath(), event.GetSize())

    body := string(event.GetBody())

    
    if len(body) == 0 {
        context.Logger.Debug("Body empty")
        return nuclio.Response{
            StatusCode:  204,
            ContentType: "application/text",
            
            Body: []byte("Body empty"),
         }, nil
    }

    myRegexPattern := string(event.GetHeader("X-Regex").([]byte))
    context.Logger.Info(myRegexPattern)
    r, err := regexp.Compile(myRegexPattern)

    if err !=  nil {
        return nuclio.Response{
            StatusCode:  400,
            ContentType: "application/text",
            Body: []byte("Regex error"),
         }, nil
    }
    
    fields := reSubMatchMap(r, body)
    
	if fields != nil { 
        // Format into JSON 
 		fieldsJson, _ := json.Marshal(fields) 	
	    return nuclio.Response{
	           StatusCode: 200,
	           ContentType: "application/json",
	           Body: []byte(fieldsJson),
	    }, nil
 	} else {
	    // Catch non matches and return 204
        context.Logger.Debug("Body empty")
        return nuclio.Response{
            StatusCode:  204,
            ContentType: "application/text",
            Body: []byte("Body empty"),
         }, nil
	}
}

func reSubMatchMap(r *regexp.Regexp, str string) (map[string]string) {
    
    match := r.FindStringSubmatch(str)

    if match != nil {
   		subMatchMap := make(map[string]string)
    	for i, name := range r.SubexpNames() {
        	if i != 0 {
            	subMatchMap[name] = match[i]
        	}
    	} 	
	return subMatchMap
     
    } else {
    	return nil
    }
}
