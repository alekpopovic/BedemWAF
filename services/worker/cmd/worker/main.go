package main

import (
	"fmt"
	"os"
)

const (
	serviceName = "bedemwaf-worker"
	version     = "dev"
)

func main() {
	fmt.Fprintf(os.Stdout, "%s %s\n", serviceName, version)
	fmt.Fprintln(os.Stdout, "TODO: start async job runner for rule updates, event enrichment, and retention cleanup")
}
