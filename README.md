# Experimental Nuclio Event ETL Functions

- ** Author **: Mika Borner <mika.borner@gmail.com>
- ** Description**:    ETL Functions for Extracting, Loading and Transforming timeseries events
- ** Version **:    @build.version@

## Functions

### fieldextractor

- Header: X-Regex: *Regex*
- Body: *String*
- Response: *JSON with fields*

- Example:

```bash
$ curl localhost:42314/ \
  -H "X-Regex: Name=(?P<field1>\w+).*?Firstname=(?P<field2>\w+)" \
  -d "Name=Kent Firstname=Clark" \
  -vvvv

   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 42314 (#0)
> POST / HTTP/1.1
> Host: localhost:42314
> User-Agent: curl/7.47.0
> Accept: */*
> X-Regex: Name=(?P<field1>\w+).*?Firstname=(?P<field2>\w+)
> Content-Length: 25
> Content-Type: application/x-www-form-urlencoded
>
* upload completely sent off: 25 out of 25 bytes
< HTTP/1.1 200 OK
< Server: nuclio
< Date: Tue, 27 Feb 2018 15:10:43 GMT
< Content-Type: application/json
< Content-Length: 34
<
* Connection #0 to host localhost left intact
{"field1":"Kent","field2":"Clark"}
```