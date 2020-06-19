// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAlertmanagerConfigHandler(t *testing.T) {

	vmiName = "vmi-test"
	namespace = "vmi-test"
	testConfig := "vmi-" + vmiName + "-alertmanager-config"

	testclient := newAlertManagerTestClient(t, vmiName, namespace, testConfig)

	// Retrieve the current version
	req, err := http.NewRequest("GET", "/alertmanager/config", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetAlertmanagerConfig)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "changemeNOW")

	// Supply a bad timestamp
	req, err = http.NewRequest("GET", "/alertmanager/config?version=badtimestamp", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerConfig)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusBadRequest, "ERROR: The version timestamp provided is not valid.")

	// Valid timestamp, but no such older version
	req, err = http.NewRequest("GET", "/alertmanager/config?version=2018-01-02T15-09-09", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerConfig)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusNotFound, "Unable to find the requested Alertmanager configuration.")

	// Retrieve a legit older version
	req, err = http.NewRequest("GET", "/alertmanager/config?version=2016-05-02T15-04-05", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerConfig)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusOK, "myconfig2")
}

func TestGetAlertmanagerVersionsHandler(t *testing.T) {

	vmiName = "vmi-test"
	namespace = "vmi-test"
	testConfig := "vmi-" + vmiName + "-alertmanager-config"
	expectedOutput := `{
	"versions": [
		"2019-07-02T15-04-05",
		"2017-02-02T15-04-05",
		"2016-05-02T15-04-05"
	]
}`
	testclient := newAlertManagerTestClient(t, vmiName, namespace, testConfig)

	req, err := http.NewRequest("GET", "/alertmanager/config/versions", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetAlertmanagerVersions)
	handler.ServeHTTP(rr, req)
	verify(t, rr, http.StatusOK, expectedOutput)
}

// ##############################################################################################
//  ALERT MANAGER HANDLER TEST UTILITIES
// ##############################################################################################

// Used by all amh unit tests to create a test clientset
func newAlertManagerTestClient(t *testing.T, vmiName string, namespace string, configName string) *K8s {
	testclient := K8s{}

	versionsConfigName := configName + "-versions"

	testMapPath := "alertmanager.yml"
	testDataString := `
route:
  receiver: "test-orig"
  group_by: ['alertname']
  group_wait: 30s
  group_interval: 1m
  repeat_interval: 3m
receivers:
- name: "test-orig"
  pagerduty_configs:
  - service_key: changemeNOW`

	// Create some fake data for the versions configMap
	versionsMap := getDefaultAlertmanagerVersionsMap(testMapPath)

	mainConfigMap := getTestConfigMap(configName, namespace, testMapPath, testDataString)
	versionsConfigMap := getTestConfigMapFromMap(versionsConfigName, namespace, versionsMap)

	testclient.ClientSet = k8sfake.NewSimpleClientset(mainConfigMap, versionsConfigMap)

	fakeVMIJson := gabs.New()
	fakeVMIJson.SetP(fmt.Sprintf("%s/%s", VMIGroup, VMIVersion), "apiVersion")
	fakeVMIJson.SetP("VMI", "kind")
	fakeVMIJson.SetP(vmiName, "name")
	fakeVMIJson.SetP(vmiName, VMIMetadataNamePath)
	fakeVMIJson.SetP(namespace, "namespace")
	fakeVMIJson.SetP(configName, AlertmanagerConfigMapPath)
	fakeVMIJson.SetP(versionsConfigName, AlertmanagerVersionsConfigMapPath)

	testServer, _, _ := getTestServerEnv(t, fakeVMIJson.String())

	c, err := newRestClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testclient.RestClient = c
	return &testclient
}

// Create some fake backup configurations
func getDefaultAlertmanagerVersionsMap(keyName string) map[string]string {
	key1 := keyName + "-2017-02-02T15-04-05"
	key2 := keyName + "-2016-05-02T15-04-05"
	key3 := keyName + "-2019-07-02T15-04-05"
	value1 := "myconfig1"
	value2 := "myconfig2"
	value3 := "myconfig3"

	return map[string]string{key1: value1, key2: value2, key3: value3}
}
