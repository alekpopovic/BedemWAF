package main

import (
	"fmt"
	"os"
)

const (
	serviceName = "bedemwaf-control-api"
	version     = "dev"
)

func main() {
	fmt.Fprintf(os.Stdout, "%s %s\n", serviceName, version)
	fmt.Fprintln(os.Stdout, "TODO: start REST API, validation, database access, auth, and OpenAPI documentation")
}
