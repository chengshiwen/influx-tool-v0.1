# InfluxDB Tool

Tool for InfluxDB and Influx Proxy

## Usage

```
$ ./influx-tool -h

Usage of ./influx-tool:
  -boolean-fields string
        fields required to cast to boolean from string, split by ','
  -database string
        database to connect to the server
  -dir string
        directory to export (default "export")
  -end string
        the end unix time to export (second precision), optional
  -float-fields string
        fields required to cast to float from string, split by ','
  -format string
        the output format to export, valid values are line or csv (default "line")
  -host string
        host to connect to (default "127.0.0.1")
  -integer-fields string
        fields required to cast to integer from string, split by ','
  -measurements string
        measurements split by ',' while return all measurements if empty
        wildcard '*' and '?' supported
  -merge
        merge and export into one file, ignored when -format is not line
  -password string
        password to connect to the server
  -port int
        port to connect to (default 8086)
  -range string
        measurements range to export, as 'start,end', started from 1, included end
        ignored when -measurements not empty
  -ssl
        use https for requests
  -start string
        the start unix time to export (second precision), optional
  -username string
        username to connect to the server
  -version
        display the version and exit
  -worker int
        number of concurrent workers to export (default 1)
```

[Chinese Tutorial](docs/tutorial.md)
