// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// +build integration

package handlers

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var amToolPath = flag.String("amToolPath", "/opt/tools/bin/amtool", "full path to amtool binary")

func TestPutAlertmanagerConfigHandler(t *testing.T) {

	vmiName = "vmi-am-test"
	namespace = "vmi-am-test"
	testConfig := "vmi-" + vmiName + "-alertmanager-config"

	amtoolPath = *amToolPath

	testBody1 := `route:
  receiver: changed-receiver
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 3m
receivers:
- name: changed-receiver
  pagerduty_configs:
  - service_key: x9ggtv339564940uvvgt8555`

	testBody2 := `route:
  receiver: changed-receiver
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 3m
receivers:
- name: changed-receiver
  pagerduty_configs:
  - service_key: x9ggtv339564940uvvgt8557`

	testclient := newAlertManagerTestClient(t, vmiName, namespace, testConfig)

	// Pre-check - should be three backups in alertmanager-config-versions
	req, err := http.NewRequest("GET", "/alertmanager/config/versions", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetAlertmanagerVersions)
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify we get three versions files back
	var versionsMap map[string][]string
	err = json.Unmarshal([]byte(rr.Body.String()), &versionsMap)
	if len(versionsMap["versions"]) != 3 {
		t.Errorf("handler returned an unexpected number of versions: expected 3. Output: %v", versionsMap["versions"])
	}

	// Run PUT with no body provided
	req, err = http.NewRequest("PUT", "/alertmanager/config", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusBadRequest, "Failed to validate config with amtool")

	// Run PUT with invalid body
	req, err = http.NewRequest("PUT", "/alertmanager/config", strings.NewReader("random string"))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusBadRequest, "Failed to validate config with amtool")

	// Run the PUT command
	req, err = http.NewRequest("PUT", "/alertmanager/config", strings.NewReader(testBody1))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusAccepted, "The Alertmanager configuration is being updated.")

	// Quick pause for update - update takes a moment
	time.Sleep(2 * time.Second)

	// Verify a new key was added to alertmanager-config-versions - should now be four
	req, err = http.NewRequest("GET", "/alertmanager/config/versions", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerVersions)
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	// Verify we get four version files back
	err = json.Unmarshal([]byte(rr.Body.String()), &versionsMap)
	if len(versionsMap["versions"]) != 4 {
		t.Errorf("handler returned an unexpected number of versions: expected 4. Output: %v", versionsMap["versions"])
	}

	// Verify the alertmanager-config ConfigMap was updated succcessfully
	req, err = http.NewRequest("GET", "/alertmanager/config", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "x9ggtv339564940uvvgt8555")

	// Check for idempotency in PUT
	req, err = http.NewRequest("PUT", "/alertmanager/config", strings.NewReader(testBody1))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusOK, "The provided body is identical to the current AlertManager configuration. No action will be taken.")

	// Run the PUT command
	req, err = http.NewRequest("PUT", "/alertmanager/config", strings.NewReader(testBody2))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusAccepted, "The Alertmanager configuration is being updated.")

	// Quick pause for update - the update takes a moment
	time.Sleep(2 * time.Second)

	// Verify a new key was added to alertmanager-config-versions - should now be five
	req, err = http.NewRequest("GET", "/alertmanager/config/versions", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerVersions)
	handler.ServeHTTP(rr, req)

	err = json.Unmarshal([]byte(rr.Body.String()), &versionsMap)
	if len(versionsMap["versions"]) != 5 {
		t.Errorf("handler returned an unexpected number of versions: expected 5. Output: %v", versionsMap["versions"])
	}

	// Verify the alertmanager-config ConfigMap was updated succcessfully
	req, err = http.NewRequest("GET", "/alertmanager/config", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "x9ggtv339564940uvvgt8557")

}

func TestPutBadAlertmanagerConfig(t *testing.T) {

	vmiName = "vmi-test"
	namespace = "vmi-test"
	testConfig := "vmi-" + vmiName + "-alertmanager-config"

	amtoolPath = "/opt/tools/bin/amtool"

	testBody := `route:
  receiver: test
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 3m`

	testclient := newAlertManagerTestClient(t, vmiName, namespace, testConfig)

	// Send an invalid configuration
	req, err := http.NewRequest("PUT", "/alertmanager/config", strings.NewReader(testBody))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.PutAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusBadRequest, "Failed to validate config with amtool:")
}
