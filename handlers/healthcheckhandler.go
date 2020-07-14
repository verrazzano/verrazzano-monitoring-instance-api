// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"net/http"
)

// /healthcheck router function
func GetHealthCheck(w http.ResponseWriter, r *http.Request) {
	//Only print to stdout in Debug Mode so it does not flood the Pod logs since k8s calls this API every 5 secs for liveness probes.
	log(LevelDebug, "200 OK: %s", "Healthy")
	w.WriteHeader(200)
	// Write output in json Format has per API healthcheck response best practices.
	w.Write([]byte("{\"description\": \"Health of Verrazzano Monitoring Instance API service\", \"status\": \"pass\"}"))
}
