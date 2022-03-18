# logging

[![GoDoc](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/github.com/smallstep/logging)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Logging is a collection of collection of packages to make easy the logging experience.

Logging is currently in active development and it's API WILL CHANGE WITHOUT NOTICE.

## logging/httplog - the HTTP middleware

Httplog implements and HTTP middleware that can optionally log the raw request and responses.

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/smallstep/logging"
    "github.com/smallstep/logging/httplog"
)

func main() {
    logger, err := logging.New("scim",
        logging.WithLogResponses(),
        logging.WithLogRequests(),
    )
    if err != nil {
        log.Fatal(err)
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello World!")
    })

    srv := &http.Server{
        Addr:    ":8080",
        Handler: httplog.Middleware(logger, http.DefaultServeMux),
        ErrorLog: logger.StdLogger(logging.ErrorLevel),
    }

    logger.Infof("start listening at %s.", srv.Addr)
    log.Fatal(srv.ListenAndServe())
}
```

A simple `curl http://localhost:8080` will print the following logs:

```sh
$ go run examples/httplog.go
{"level":"info","ts":1582944317.382856,"caller":"logging/logger.go:87","msg":"start listening at :8080."}
{"level":"info","ts":1582944330.861353,"caller":"httplog/handler.go:121","msg":"","name":"scim","request-id":"bpct0ijipt3avli7utp0","remote-address":"::1","time":"2020-02-28T18:45:30-08:00","duration":0.000064785,"duration-ns":64785,"method":"GET","path":"/","protocol":"HTTP/1.1","status":200,"size":13,"referer":"","user-agent":"curl/7.64.1","request":"R0VUIC8gSFRUUC8xLjENCkhvc3Q6IGxvY2FsaG9zdDo4MDgwDQpBY2NlcHQ6ICovKg0KVXNlci1BZ2VudDogY3VybC83LjY0LjENClgtVHJhY2UtSWQ6IGJwY3QwaWppcHQzYXZsaTd1dHAwDQoNCg==","response":"SGVsbG8gV29ybGQhCg=="}
```
