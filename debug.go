package main

import (
	"log"
	"os"
)

// debugEnabled enables verbose per-key and focus diagnostics. It is off by
// default so normal typing does not spam the journal or add latency.
var debugEnabled = func() bool {
	v := os.Getenv("BAMBOO_DEBUG")
	return v != "" && v != "0"
}()

func debugf(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf(format, args...)
	}
}
