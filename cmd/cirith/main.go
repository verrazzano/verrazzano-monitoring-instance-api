// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// Verrazzano Monitoring Instance API Server
//
// .
//
//     BasePath: /
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//
// swagger:meta
package main

import (
	"fmt"
	"github.com/rs/zerolog"
	handler "github.com/verrazzano/verrazzano-monitoring-instance-api/handlers"
	"net/http"
	"os"
	"strconv"
)

func main() {

	// initialize logs before execution
	InitLogs()

	// get config
	config := handler.GetConfig()

	k8s, err := handler.NewK8s(config)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Create a New Router with all handlers
	router := k8s.NewRouter(config)

	// Start the server
	http.ListenAndServe(handler.ListenURL, router)
	fmt.Fprintf(os.Stdout, "quit unexpectedly: reason unknown")

}

// Initialize logs with Time and Global Level of Logs set at Info
func InitLogs() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Log levels are outlined as follows:
	// Panic: 5
	// Fatal: 4
	// Error: 3
	// Warn: 2
	// Info: 1
	// Debug: 0
	// Trace: -1
	// more info can be found at https://github.com/rs/zerolog#leveled-logging

	envLog := os.Getenv("LOG_LEVEL")
	if val, err := strconv.Atoi(envLog); envLog != "" && err == nil && val >= -1 && val <= 5 {
		zerolog.SetGlobalLevel(zerolog.Level(val))
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}