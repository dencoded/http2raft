package main

import (
	"flag"

	"github.com/dencoded/http2raft"
)

var httpAddr = flag.String("http_addr", "", "host address to run HTTP server on")

func main() {
	flag.Parse()

	if *httpAddr == "" {
		panic("'http_addr' parameter is missing")
	}

	http2raft.Start(*httpAddr, NewInMemCounter)
}
