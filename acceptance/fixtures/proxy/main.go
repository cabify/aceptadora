package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// This is a dummy reverse proxy to illustrate how our services can access the acceptance tester
func main() {
	port := os.Getenv("PROXY_PORT")
	targetURLRaw := os.Getenv("PROXY_TARGETURL")

	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(rw http.ResponseWriter, _ *http.Request) {
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
