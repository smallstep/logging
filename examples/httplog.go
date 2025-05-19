package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/smallstep/logging"
	"github.com/smallstep/logging/httplog"
)

func main() {
	logger, err := logging.New("saluter",
		logging.WithLogResponses(),
		logging.WithLogRequests(),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Hello World!")
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           httplog.Middleware(logger, http.DefaultServeMux),
		ErrorLog:          logger.StdLogger(logging.ErrorLevel),
		ReadHeaderTimeout: 30 * time.Second,
	}

	logger.Infof("start listening at %s.", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
