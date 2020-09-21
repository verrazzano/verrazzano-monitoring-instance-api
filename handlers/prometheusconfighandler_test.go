// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jeffail/gabs/v2"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestGetPrometheusConfigHandler(t *testing.T) {

	vmiName = "vmi-prom-test"
	namespace = "vmi-prom-test"
	testclient := newPrometheusConfigTestClient(t, vmiName, namespace)

	// Retrieve the current version
	req, err := http.NewRequest("GET", "/prometheus/config", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetPrometheusConfig)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusOK, "__meta_kubernetes_pod_annotation_fakekdev_io_scrape")

	// Supply a bad timestamp
	req, err = http.NewRequest("GET", "/prometheus/config?version=bob", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusConfig)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusBadRequest, "ERROR: The version timestamp provided is not valid.")

	// Valid timestamp, but no such older version
	req, err = http.NewRequest("GET", "/prometheus/config?version=2018-01-02T15-09-09", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusConfig)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusNotFound, "Unable to find the requested Prometheus configuration.")

	// Retrieve an older version
	req, err = http.NewRequest("GET", "/prometheus/config?version=2018-01-02T15-04-05", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusConfig)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusOK, "myconfig1")
}

func TestGetPrometheusVersionsHandler(t *testing.T) {

	vmiName = "vmi-prom-test"
	namespace = "vmi-prom-test"
	expectedOutput := `{
	"versions": [
		"2019-05-02T15-04-05",
		"2018-01-02T15-04-05",
		"2016-02-02T15-04-05"
	]
}`
	testclient := newPrometheusConfigTestClient(t, vmiName, namespace)

	req, err := http.NewRequest("GET", "/prometheus/config/versions", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetPrometheusVersions)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusOK, expectedOutput)
}

// ##############################################################################################
//  PROMETHEUS CONFIG HANDLER TEST UTILITIES
// ##############################################################################################

// Used by all pch unit tests to create a test clientset
func newPrometheusConfigTestClient(t *testing.T, vmiName string, namespace string) *K8s {

	mainConfigMapName := "vmi-" + vmiName + "-prometheus-config"
	versionsConfigMapName := "vmi-" + vmiName + "-prometheus-config-versions"
	testclient := K8s{}

	mainConfigKey := "prometheus.yml"
	mainConfigValue := `# fake global config
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
   scrape_interval: 15s
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
 - job_name: fake_dev
   honor_labels: true
   params:
     match[]:
     - '{__name__=~".+"}'
   scrape_interval: 120s
   scrape_timeout: 15s
   metrics_path: /federate
   scheme: http
   static_configs:
   - targets:
     - fake-dev.example.com:19090
     labels:
       env: fake_metrics`

	// Create an initial map for the versions configMap
	versionsMap := getDefaultPrometheusVersionsMap(mainConfigKey)

	// Create the configMaps
	mainConfigMap := getTestConfigMap(mainConfigMapName, namespace, mainConfigKey, mainConfigValue)
	versionsConfigMap := getTestConfigMapFromMap(versionsConfigMapName, namespace, versionsMap)

	testclient.ClientSet = k8sfake.NewSimpleClientset(mainConfigMap, versionsConfigMap)

	fakeVMIJson := gabs.New()
	fakeVMIJson.SetP(fmt.Sprintf("%s/%s", VMIGroup, VMIVersion), "apiVersion")
	fakeVMIJson.SetP("VMI", "kind")
	fakeVMIJson.SetP(vmiName, "name")
	fakeVMIJson.SetP(vmiName, VMIMetadataNamePath)
	fakeVMIJson.SetP(namespace, "namespace")
	fakeVMIJson.SetP(mainConfigMapName, PrometheusConfigMapPath)
	fakeVMIJson.SetP(versionsConfigMapName, PrometheusVersionsConfigMapPath)

	testServer, _, _ := getTestServerEnv(t, fakeVMIJson.String())

	c, err := newRestClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testclient.RestClient = c
	return &testclient
}

// Create some fake backup versions
func getDefaultPrometheusVersionsMap(keyName string) map[string]string {
	key1 := keyName + "-2018-01-02T15-04-05"
	key2 := keyName + "-2016-02-02T15-04-05"
	key3 := keyName + "-2019-05-02T15-04-05"
	value1 := "myconfig1"
	value2 := "myconfig2"
	value3 := "myconfig3"

	return map[string]string{key1: value1, key2: value2, key3: value3}
}
