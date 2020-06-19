// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// +build integration

package handlers

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateUpdateDeletePrometheusNamedRules(t *testing.T) {
	vmiName = "vmi-prometheus-rules-test"
	namespace = "vmi-prometheus-rules-test"
	promtoolPath = "/opt/tools/bin/promtool"

	testConfig := "vmi-" + vmiName + "-prometheus-config"
	testRules1InvalidBody := `groups:
- name: HighErrorRate
  rules:
  - alert: HighErrorRate
    expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
    for: 10m
    labels:
      severity: page
    abc: def
    annotations:
      summary: High request latency`

	testRules1Body := `groups:
- name: HighErrorRate
  rules:
  - alert: HighErrorRate
    expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
    for: 10m
    labels:
      severity: page
    annotations:
      summary: High request latency`

	testRules2Body := `groups:
- name: example
  rules:

  # Alert for any instance that is unreachable for >5 minutes.
  - alert: InstanceDown
    expr: up == 0
    for: 5m
    labels:
      severity: page
    annotations:
      summary: "Instance {{ $labels.instance }} down"
      description: "{{ $labels.instance }} of job {{ $labels.job }} has been down for more than 5 minutes."`
	testRules2UpdatedBody := `groups:
- name: example
  rules:

  # Alert for any instance that is unreachable for >5 minutes.
  - alert: InstanceDown
    expr: up == 0
    for: 2m
    labels:
      severity: page
    annotations:
      summary: "Instance {{ $labels.instance }} down"
      description: "{{ $labels.instance }} of job {{ $labels.job }} has been down for more than 5 minutes."`

	testclient := newRulesTestClient(t, vmiName, namespace, testConfig)

	/* *** Get list of Prometheus alert rules - should be none *** */
	req, err := http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "No Prometheus alert rules were found.")

	/* *** Request a non-existent current file *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/noSuch.rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRules)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusNotFound, "Unable to find a current Prometheus Alert rule called: noSuch.rules")

	/* *** Create a prometheus alert rules file without .rules extension *** */
	testRules1 := "testrules1.xyz"

	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules1, strings.NewReader(testRules1Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusBadRequest, "File name must end with: .rules")

	/* *** Create a prometheus alert rules file invalid yaml file *** */
	testRules1 = "testrules1.rules"

	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules1, strings.NewReader(testRules1InvalidBody))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusBadRequest, "Failed to validate with promtool:")

	/* *** Create a prometheus alert rules file *** */
	testRules1 = "testrules1.rules"

	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules1, strings.NewReader(testRules1Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal := "A new rule file: " + testRules1 + " is being created."
	verify(t, rr, http.StatusAccepted, expectedReturnVal)

	/* *** Try to create the same rule with no changes to the body *** */
	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules1, strings.NewReader(testRules1Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = "The provided body is identical to the current Alert Rule: " + testRules1 + ". No action will be taken."
	verify(t, rr, http.StatusOK, expectedReturnVal)

	/* *** Get All Rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testRules1)

	/* *** Get versions of testrules1.rules - there aren't any *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+testRules1+"/versions", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleVersions)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "No older versions of the Prometheus alert rule were found.")

	/* *** Get testRules1 *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+testRules1, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRules)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testRules1Body)

	/* *** Create a testRule2 *** */
	testRules2 := "testrules2.rules"
	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules2, strings.NewReader(testRules2Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = "A new rule file: " + testRules2 + " is being created."
	verify(t, rr, http.StatusAccepted, expectedReturnVal)

	/* *** Get All Rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	// Verify both rules files appear
	verify(t, rr, http.StatusOK, testRules1)
	verify(t, rr, http.StatusOK, testRules2)

	/* *** Get testRules2 *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+testRules2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = strings.TrimSpace(testRules2Body)
	verify(t, rr, http.StatusOK, expectedReturnVal)

	/* *** Update testRules2 *** */
	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules2, strings.NewReader(testRules2UpdatedBody))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = "The existing rule: " + testRules2 + " is being updated."
	verify(t, rr, http.StatusAccepted, expectedReturnVal)

	/* *** Get All Rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	// Verify both rules files appear
	verify(t, rr, http.StatusOK, testRules1)
	verify(t, rr, http.StatusOK, testRules2)

	/* *** Get testRules2 *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+testRules2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = testRules2UpdatedBody
	verify(t, rr, http.StatusOK, expectedReturnVal)

	/* *** Delete testRules1 *** */
	req, err = http.NewRequest("DELETE", "/prometheus/rules/"+testRules1, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.DeletePrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = "The current alert rule: " + testRules1 + " and all older versions are being deleted."
	verify(t, rr, http.StatusAccepted, expectedReturnVal)

	/* *** Get testRules1 *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+testRules1, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRules)
	handler.ServeHTTP(rr, req)

	verifyStatus(t, rr, http.StatusNotFound)

	/* *** Get All Rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = testRules2
	verify(t, rr, http.StatusOK, expectedReturnVal)

	/* *** Delete testRules2 *** */
	req, err = http.NewRequest("DELETE", "/prometheus/rules/"+testRules2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.DeletePrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = "The current alert rule: " + testRules2 + " and all older versions are being deleted."
	verify(t, rr, http.StatusAccepted, expectedReturnVal)

	/* *** Get testRules2 *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+testRules2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRules)
	handler.ServeHTTP(rr, req)

	verifyStatus(t, rr, http.StatusNotFound)

	/* *** GetAllRules *** */
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, "No Prometheus alert rules were found.")

	/* *** Get All Older Rules for testRules1- should be none *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+testRules1+"/versions", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleVersions)
	handler.ServeHTTP(rr, req)

	// Verify no files are returned
	expectedReturnVal = "ERROR: Unable to find a current Prometheus Alert rule called: " + testRules1
	verify(t, rr, http.StatusNotFound, expectedReturnVal)
}

func TestCreateUpdateDeletePrometheusUnnamedRules(t *testing.T) {
	vmiName = "vmi-prometheus-rules-test"
	namespace = "vmi-prometheus-rules-test"
	promtoolPath = "/opt/tools/bin/promtool"

	testConfig := "vmi-" + vmiName + "-prometheus-config"

	testRules1Body := `groups:
- name: HighErrorRate
  rules:
  - alert: HighErrorRate
    expr: job:request_latency_seconds:mean5m{job="myjob"} > 0.5
    for: 10m
    labels:
      severity: page
    annotations:
      summary: High request latency`

	testRules2Body := `groups:
- name: example
  rules:

  # Alert for any instance that is unreachable for >5 minutes.
  - alert: InstanceDown
    expr: up == 0
    for: 5m
    labels:
      severity: page
    annotations:
      summary: "Instance {{ $labels.instance }} down"
      description: "{{ $labels.instance }} of job {{ $labels.job }} has been down for more than 5 minutes."`

	testclient := newRulesTestClient(t, vmiName, namespace, testConfig)

	/* *** Get All Rules *** */
	req, err := http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		fmt.Println(rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	/* *** Create a named prometheus alert rules file *** */
	testRules1 := "testrules1.rules"

	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules1, strings.NewReader(testRules1Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusAccepted)
	}

	expectedReturnVal := "A new rule file: " + testRules1 + " is being created."
	if strings.TrimSpace(rr.Body.String()) != expectedReturnVal {
		t.Errorf("handler returned unexpected body: Expected '%s' in response but found %v", expectedReturnVal, rr.Body.String())
	}

	/* *** Get All Rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testRules1)

	/* *** Create a another named rule testRule2 *** */
	testRules2 := "testrules2.rules"
	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules2, strings.NewReader(testRules2Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusAccepted)
	}

	expectedReturnVal = "A new rule file: " + testRules2 + " is being created."
	if strings.TrimSpace(rr.Body.String()) != expectedReturnVal {
		t.Errorf("handler returned unexpected body: Expected '%s' in response but found %v", expectedReturnVal, rr.Body.String())
	}

	/* *** Get All Rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		fmt.Println(rr.Body.String())

		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify both rules files appear
	verify(t, rr, http.StatusOK, testRules1)
	verify(t, rr, http.StatusOK, testRules2)

	/* *** Create an *unnamed* named rule, which will return a proper error  *** */
	req, err = http.NewRequest("PUT", "/prometheus/rules", strings.NewReader(testRules2Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusUnnamedRules)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusBadRequest, "This endpoint has been deprecated in Cirith v1.\nPlease use:  PUT /prometheus/rules/"+unnamedRules)

	/* *** Try to get unnamed.rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules/"+unnamedRules, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRules)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusNotFound, "Unable to find a current Prometheus Alert rule called: "+unnamedRules)

	/* *** Create an alert rule file called "unnamed.rules" *** */
	testRules := unnamedRules
	req, err = http.NewRequest("PUT", "/prometheus/rules/"+testRules, strings.NewReader(testRules2Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutPrometheusRules)
	handler.ServeHTTP(rr, req)

	expectedReturnVal = "A new rule file: " + testRules + " is being created."
	verify(t, rr, http.StatusAccepted, expectedReturnVal)

	/* *** Get All Rules *** */
	req, err = http.NewRequest("GET", "/prometheus/rules", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetPrometheusRuleNames)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		fmt.Println(rr.Body.String())
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify the unnamed.rules file appears
	verify(t, rr, http.StatusOK, unnamedRules)
}

func newRulesTestClient(t *testing.T, vmiName string, namespace string, configName string) *K8s {
	testclient := K8s{}

	versionsConfigName := configName + "-versions"

	mainConfigMap := createEmptyTestConfigMap(configName, namespace)
	versionsConfigMap := createEmptyTestConfigMap(versionsConfigName, namespace)

	testclient.ClientSet = k8sfake.NewSimpleClientset(mainConfigMap, versionsConfigMap)

	fakeVMIJson := gabs.New()
	fakeVMIJson.SetP(fmt.Sprintf("%s/%s", VMIGroup, VMIVersion), "apiVersion")
	fakeVMIJson.SetP("VMI", "kind")
	fakeVMIJson.SetP(vmiName, "name")
	fakeVMIJson.SetP(vmiName, VMIMetadataNamePath)
	fakeVMIJson.SetP(namespace, "namespace")
	fakeVMIJson.SetP(configName, PrometheusRulesConfigMapPath)
	fakeVMIJson.SetP(versionsConfigName, PrometheusRulesVersionsConfigMapPath)

	testServer, _, _ := getTestServerEnv(t, fakeVMIJson.String())

	c, err := newRestClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testclient.RestClient = c
	return &testclient
}
