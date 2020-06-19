// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// heahthcheckhandlers_test.go
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheckHandler(t *testing.T) {

	req, err := http.NewRequest("GET", "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetHealthCheck)

	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "{\"description\": \"Health of Verrazzano Monitoring Instance API service\", \"status\": \"pass\"}")
}
