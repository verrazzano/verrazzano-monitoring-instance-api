// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
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
	handler "github.com/verrazzano/verrazzano-monitoring-instance-api/handlers"
	"net/http"
	"os"
)

func main() {

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
