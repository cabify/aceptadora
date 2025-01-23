package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

// This is a dummy reverse proxy to illustrate how our services can access the acceptance tester
func main() {
	port := os.Getenv("PROXY_PORT")
	targetURLRaw := os.Getenv("PROXY_TARGETURL")
	startedAt := time.Now()

	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(rw http.ResponseWriter, _ *http.Request) {
		_, err := rw.Write([]byte(fmt.Sprintf("{\"started_at\": %v}", startedAt.UnixMilli())))
		if err != nil {
			return
		}
		rw.WriteHeader(http.StatusOK)
	})

	targetURL, err := url.Parse(targetURLRaw)
	if err != nil {
		panic(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
	mux.Handle("/", reverseProxy)

	log.Printf("Starting proxy on 0.0.0.0:%s for %s", port, targetURLRaw)
	http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
}
