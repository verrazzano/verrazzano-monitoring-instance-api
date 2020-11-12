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
	"flag"
	"fmt"
	"net/http"
	"os"

	handler "github.com/verrazzano/verrazzano-monitoring-instance-api/handlers"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	zapOptions := kzap.Options{}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()
	InitLogs(zapOptions)

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

// InitLogs initializes logs with Time and Global Level of Logs set at Info
func InitLogs(opts kzap.Options) {
	var config zap.Config
	if opts.Development {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	if opts.Level != nil {
		config.Level = opts.Level.(zap.AtomicLevel)
	} else {
		config.Level.SetLevel(zapcore.InfoLevel)
	}
	config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	config.EncoderConfig.TimeKey = "@timestamp"
	config.EncoderConfig.MessageKey = "message"
	logger, err := config.Build()
	if err != nil {
		zap.S().Errorf("Error creating logger %v", err)
	} else {
		zap.ReplaceGlobals(logger)
	}
}
