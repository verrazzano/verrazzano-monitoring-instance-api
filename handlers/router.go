// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"github.com/gorilla/mux"
	restgo "k8s.io/client-go/rest"
	"net/http"
)

// Verrazzano Monitoring Instance Cirith Server Swagger
var permanentRedirects = map[string]string{
	"/prometheus/resize/halve":     "/v1",
	"/prometheus/resize/double":    "/v1",
}

// Verrazzano Monitoring Instance Cirith Server Swagger
func (k *K8s) NewRouter(config *restgo.Config) *mux.Router {

	router := mux.NewRouter().StrictSlash(true)

	// Set root content to Swagger docs
	router.Handle("/", http.FileServer(http.Dir(staticPath)))

	for path, redirect := range permanentRedirects {
		router.Handle(path, redirectHandler(redirect))
	}

	router.HandleFunc("/healthcheck", GetHealthCheck).Methods("GET")

	// swagger:operation GET /prometheus/config getPrometheusConfig
	// ---
	// tags:
	// - "Prometheus Config"
	// summary: Display the contents of a Prometheus configuration file.
	// description: Display the contents of the current Prometheus configuration file.  If a version parmaeter is provided, display the contents of that older version.
	// parameters:
	// - in: query
	//   name: version
	//   description: Timestamp of older file version
	//   required: false
	//   schema:
	//     type: string
	// responses:
	//   "200":
	//     description: Display the contents of a Prometheus config file
	router.HandleFunc("/prometheus/config", k.GetPrometheusConfig).Methods("GET")

	// swagger:operation PUT /prometheus/config putPrometheusConfig
	// ---
	// tags:
	// - "Prometheus Config"
	// summary: Replace contents of the Prometheus configuration file.
	// description:  The user-provided content will replace the current Prometheus configuration.  The older configuration is saved to a file.
	// consumes:
	// - application/x-yaml
	// parameters:
	// - in: body
	//   name: body
	//   description: New contents of the Prometheus config file
	//   required: true
	//   schema:
	//     type: string
	// responses:
	//   "200":
	//     description: Replace contents of Prometheus config file (as specified by -promConfigFile)
	router.HandleFunc("/prometheus/config", k.PutPrometheusConfig).Methods("PUT")

	// swagger:operation GET /prometheus/config/versions getPrometheusVersions
	// ---
	// tags:
	// - "Prometheus Config"
	// summary: Display a list of all older saved versions.
	// description: Display a list of all older saved versions of the Prometheus configuration.
	// responses:
	//   "200":
	//     description: Display a list of all older saved versions of the Prometheus configuration.
	router.HandleFunc("/prometheus/config/versions", k.GetPrometheusVersions).Methods("GET")

	//Prometheus Rules Routes
	// swagger:operation GET /prometheus/rules Prometheus Rules getPrometheusRuleNames
	// ---
	// tags:
	// - "Prometheus Alert Rules"
	// summary: Display a list of all current Prometheus Alert Rule files.
	// description: Display a list of all current Prometheus Alert Rule files.
	// responses:
	//   "200":
	//     description: Display a list of all current Prometheus Alert Rules files.
	router.HandleFunc("/prometheus/rules", k.GetPrometheusRuleNames).Methods("GET")

	// swagger:operation GET /prometheus/rules/{name}/versions getPrometheusRuleVersions
	// ---
	// tags:
	// - "Prometheus Alert Rules"
	// summary: Display a list of older versions available for a Prometheus Alert Rules file
	// description: Display a list of all older versions available for a provided Prometheus Alert Rules file
	// parameters:
	// - in: path
	//   name: name
	//   type: string
	//   required: true
	//   description: file name to search for
	// responses:
	//   "200":
	//     description: Display a list of older versions available.
	router.HandleFunc("/prometheus/rules/{name}/versions", k.GetPrometheusRuleVersions).Methods("GET")

	// PUT /prometheus/rules has been deprecated in favor of PUT /prometheus/rules/{name}
	// It has been removed from Swagger, but the endpoint will return a friendly error message
	// for the time being.
	router.HandleFunc("/prometheus/rules", k.PutPrometheusUnnamedRules).Methods("PUT")

	// swagger:operation GET /prometheus/rules/{name} getPrometheusAlertRules
	// ---
	// tags:
	// - "Prometheus Alert Rules"
	// summary: Display the contents of a Prometheus Alert Rules file.
	// description: Display the contents of a specific Prometheus Alert Rules file. If a version parameter is provided (optional), return the older version of that Alert Rules file.
	// parameters:
	// - in: path
	//   name: name
	//   type: string
	//   required: true
	//   description: Name of file
	// - in: query
	//   name: version
	//   type: string
	//   required: false
	//   description: Timestamp of older file version
	// responses:
	//   "200":
	//     description: Display contents of a Prometheus Alert Rules file.
	router.HandleFunc("/prometheus/rules/{name}", k.GetPrometheusRules).Methods("GET")

	// swagger:operation PUT /prometheus/rules/{name} putPrometheusAlertRules
	// ---
	// tags:
	// - "Prometheus Alert Rules"
	// summary: Replace contents of a current Prometheus Alert Rules file.
	// description: Update the contents of a current Prometheus Alert Rules file.  If the file already exists, a copy will be saved prior to replacement.  If the file does not currently exist, a new rules file will be created.
	// consumes:
	// - application/x-yaml
	// parameters:
	// - in: path
	//   name: name
	//   type: string
	//   required: true
	//   description: Name of file to create or update
	// - in: body
	//   name: body
	//   description: Content of the rules file to create or update.
	//   required: true
	//   schema:
	//     type: string
	// responses:
	//   "200":
	//     description: Replace contents of a current Prometheus Alert Rules file.
	router.HandleFunc("/prometheus/rules/{name}", k.PutPrometheusRules).Methods("PUT")

	// swagger:operation DELETE /prometheus/rules/{name} deletePrometheusAlertRules
	// ---
	// tags:
	// - "Prometheus Alert Rules"
	// summary: Delete a Prometheus Alert Rules file and all its older saved versions.
	// description: Delete a current Prometheus Alert Rules file and all its older saved versions. *This action cannot be undone.*
	// parameters:
	// - in: path
	//   name: name
	//   type: string
	//   required: true
	//   description: Name of file to delete
	// responses:
	//   "200":
	//     description: Delete a Prometheus Alert Rules file and all its older saved versions.
	router.HandleFunc("/prometheus/rules/{name}", k.DeletePrometheusRules).Methods("DELETE")

	router.Handle("/{rest}", http.FileServer(http.Dir(staticPath)))

	return router
}

func redirectHandler(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// POST/PUT data can be discarded on redirect by client.
		http.Redirect(w, r, target+r.RequestURI, http.StatusTemporaryRedirect)
	})
}
