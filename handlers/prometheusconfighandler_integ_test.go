// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// +build integration

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPutPrometheusConfigHandler(t *testing.T) {

	vmiName = "vmi-prom-test"
	namespace = "vmi-prom-test"
	promtoolPath = "/opt/tools/bin/promtool"

	testBody := `# fake global config
global:
  scrape_interval: 5s
  evaluation_interval: 5s
rule_files:
  - '/etc/prometheus/rules/*.rules'
alerting:
  alertmanagers:
    - static_configs:
      - targets: ["vmi-fake-dev-alertmanager:9093"]
scrape_configs:
 - job_name: 'prometheus'
   scrape_interval: 5s
   scrape_timeout: 5s
   static_configs:
   - targets: ['localhost:9090']
 - job_name: 'PushGateway'
   scrape_interval: 1500s
   scrape_timeout: 10s
   static_configs:
   - targets: ["vmi-dev-fake-prometheus-gw:9091"]
 - job_name: 'kubernetes-pods'
   kubernetes_sd_configs:
   - role: pod
     namespaces:
       names:
         - "dev-fake"      
   relabel_configs:
   - source_labels: [__meta_kubernetes_pod_annotation_fakekdev_io_scrape]
     action: keep
     regex: true
   - action: labelmap
     regex: __meta_kubernetes_pod_label_(.+)
   - source_labels: [__meta_kubernetes_namespace]
     action: replace
     target_label: kubernetes_namespace
   - source_labels: [__meta_kubernetes_pod_name]
     action: replace
     target_label: kubernetes_pod_name
 - job_name: fake_NEW_NAME_dev
   honor_labels: true
   params:
     match[]:
     - '{__name__=~".+"}'
   scrape_interval: 300s
   scrape_timeout: 15s
   metrics_path: /federate
   scheme: http
   static_configs:
   - targets:
     - fake-dev.example.com:19090
     labels:
       env: fake_metrics`

	testclient := newPrometheusConfigTestClient(t, vmiName, namespace)

	// Pre-check - should be three backups in prometheus-config-versions
	req, err := http.NewRequest("GET", "/prometheus/config/versions", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetPrometheusVersions)
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify we get three versions files back
	var versionsMap map[string][]string
	err = json.Unmarshal([]byte(rr.Body.String()), &versionsMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(versionsMap["versions"]) != 3 {
		t.Errorf("handler returned an unexpected number of versions: expected 3. Output: %v", versionsMap["versions"])
	}

	// Run PUT with no body provided
	req, err = http.NewRequest("PUT", "/prometheus/config", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusBadRequest, "Invalid Prometheus YAML: it is empty.")

	// Run PUT with invalid body
	req, err = http.NewRequest("PUT", "/prometheus/config", strings.NewReader("random string"))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusBadRequest, "Invalid Prometheus YAML: it does not have the mandatory name global.scrape_interval.")

	// Run the PUT command
	req, err = http.NewRequest("PUT", "/prometheus/config", strings.NewReader(testBody))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusAccepted, "The Prometheus configuration is being updated.")

	// Verify a new key was added to prometheus-config-versions - should now be four
	req, err = http.NewRequest("GET", "/prometheus/config/versions", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusVersions)
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify we get four version files back
	err = json.Unmarshal([]byte(rr.Body.String()), &versionsMap)
	if err != nil {
		t.Fatal(err)
	}
	if len(versionsMap["versions"]) != 4 {
		t.Errorf("handler returned an unexpected number of versions: expected 3. Output: %v", versionsMap["versions"])
	}

	// Verify the prometheus-config ConfigMap was updated succcessfully
	req, err = http.NewRequest("GET", "/prometheus/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusConfig)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "fake_NEW_NAME")

	// Check for idempotency in PUT
	req, err = http.NewRequest("PUT", "/prometheus/config", strings.NewReader(testBody))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusConfig)
	handler.ServeHTTP(rr, req)

	// Check the status code and content is what we expect.
	verify(t, rr, http.StatusOK, "The provided body is identical to the current Prometheus configuration.")
}

func TestPutBadPrometheusConfig(t *testing.T) {

	vmiName = "vmi-prom-test"
	namespace = "vmi-prom-test"
	promtoolPath = "/opt/tools/bin/promtool"

	testBody := `#new bad global config
global:
  scrape_interval:     20s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.

alerting:
  alertmanagers:
  - static_configs:
    - targets:

rule_files:
   - "alert.rules.yml"

scrape_configs:
  - job_name: 'prometheus'

    static_configs:
      - targets: ['localhost:9090']`

	testclient := newPrometheusConfigTestClient(t, vmiName, namespace)

	req, err := http.NewRequest("PUT", "/prometheus/config", strings.NewReader(testBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.PutPrometheusConfig)
	handler.ServeHTTP(rr, req)

	expectedMessage := "Error: Prometheus YAML does not have scrape_configs jobs defined."
	verify(t, rr, http.StatusBadRequest, expectedMessage)
}
