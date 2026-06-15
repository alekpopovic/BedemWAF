package main

import (
	"fmt"
	"os"
)

const (
	serviceName = "bedemwaf-gateway"
	version     = "dev"
)

func main() {
	fmt.Fprintf(os.Stdout, "%s %s\n", serviceName, version)
	fmt.Fprintln(os.Stdout, "TODO: start HTTP reverse proxy, WAF inspection, rate limiting, and audit event publishing")
}
