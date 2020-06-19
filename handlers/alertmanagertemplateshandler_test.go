// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
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

func TestCreateUpdateDeleteAlertmanagerTemplates(t *testing.T) {
	vmiName = "vmi-alertmanager-templates-test"
	namespace = "vmi-alertmanager-templates-test"

	testConfig := "vmi-" + vmiName + "-alertmanager-templates"

	testTemplates1Body := `{{ define "slack.myorg.text" }}https://internal.myorg.net/wiki/alerts/{{ .GroupLabels.app }}/{{ .GroupLabels.alertname }}{{ end}}`
	testTemplates2Body := `{{ define "email.default.html" }}<a href="http://www.google.com">ABCD</a>{{ end }}`
	testTemplates2UpdatedBody := `{{ define "email.default.html" }}<a href="http://www.google.com">EFGH</a>{{ end }}`

	testclient := newTemplatesTestClient(t, vmiName, namespace, testConfig)

	req, err := http.NewRequest("GET", "/v1/alertmanager/templates", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(testclient.GetAllAlertmanagerTemplatesFileNames)
	handler.ServeHTTP(rr, req)

	verifyStatus(t, rr, http.StatusOK)

	/* *** Create a alertmanager templates file without .tmpl extension *** */
	testTemplates1 := "testtemplates1.xyz"

	req, err = http.NewRequest("PUT", "/v1/alertmanager/template/"+testTemplates1, strings.NewReader(testTemplates1Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusBadRequest, "ERROR: Filename should end with .tmpl only")

	/* *** Create a alertmanager templates file *** */
	testTemplates1 = "testtemplates1.tmpl"

	req, err = http.NewRequest("PUT", "/v1/alertmanager/template/"+testTemplates1, strings.NewReader(testTemplates1Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusAccepted, "Adding new template file name: "+testConfig+", "+testTemplates1)

	/* *** Get All Templates *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/templates", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAllAlertmanagerTemplatesFileNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testTemplates1)

	/* *** Get testTemplates1 *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/template/"+testTemplates1, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testTemplates1Body)

	/* *** Create a testTemplate2 *** */
	testTemplates2 := "testtemplates2.tmpl"
	req, err = http.NewRequest("PUT", "/v1/alertmanager/template/"+testTemplates2, strings.NewReader(testTemplates2Body))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusAccepted, "Adding new template file name: "+testConfig+", "+testTemplates2)

	/* *** Get All Templatess *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/templates", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAllAlertmanagerTemplatesFileNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testTemplates1+"\n"+testTemplates2)

	/* *** Get testTemplates2 *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/template/"+testTemplates2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, strings.TrimSpace(testTemplates2Body))

	/* *** Update testTemplates2 *** */
	req, err = http.NewRequest("PUT", "/v1/alertmanager/template/"+testTemplates2, strings.NewReader(testTemplates2UpdatedBody))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.PutAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusAccepted, "Updating existing template in Map: "+testConfig+", "+testTemplates2)

	/* *** Get All Templatess *** */
	req, err = http.NewRequest("GET", "/secrets", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAllAlertmanagerTemplatesFileNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testTemplates1+"\n"+testTemplates2)

	/* *** Get testTemplates2 *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/template/"+testTemplates2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testTemplates2UpdatedBody)

	/* *** Delete testTemplates1 *** */
	req, err = http.NewRequest("DELETE", "/v1/alertmanager/template/"+testTemplates1, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.DeleteAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusAccepted, "Deleting template file: "+testConfig+", "+testTemplates1)

	/* *** Get testTemplates1 *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/template/"+testTemplates1, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verifyStatus(t, rr, http.StatusBadRequest)

	/* *** Get All Templatess *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/templates", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAllAlertmanagerTemplatesFileNames)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusOK, testTemplates2)

	/* *** Delete testTemplates2 *** */
	req, err = http.NewRequest("DELETE", "/v1/alertmanager/template/"+testTemplates2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.DeleteAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verify(t, rr, http.StatusAccepted, "Deleting template file: "+testConfig+", "+testTemplates2)

	/* *** Get testTemplates2 *** */
	req, err = http.NewRequest("GET", "/v1/alertmanager/template/"+testTemplates2, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAlertmanagerTemplate)
	handler.ServeHTTP(rr, req)

	verifyStatus(t, rr, http.StatusBadRequest)

	/* *** GetAllTemplatess *** */
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(testclient.GetAllAlertmanagerTemplatesFileNames)
	handler.ServeHTTP(rr, req)

	verifyStatus(t, rr, http.StatusOK)

}

func newTemplatesTestClient(t *testing.T, vmiName string, namespace string, templatesConfigName string) *K8s {
	testclient := K8s{}
	configName := "vmi-" + vmiName + "-alertmanager-config"
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
  - service_key: changemeNOW
templates:
- '/etc/alertmanager/templates/*.tmpl'`

	testConfigMap := getTestConfigMap(configName, namespace, testMapPath, testDataString)
	templatesConfigMap := createEmptyTestConfigMap(templatesConfigName, namespace)

	testclient.ClientSet = k8sfake.NewSimpleClientset(testConfigMap, templatesConfigMap)

	fakeVMIJson := gabs.New()
	fakeVMIJson.SetP(fmt.Sprintf("%s/%s", VMIGroup, VMIVersion), "apiVersion")
	fakeVMIJson.SetP("VMI", "kind")
	fakeVMIJson.SetP(vmiName, "name")
	fakeVMIJson.SetP(vmiName, VMIMetadataNamePath)
	fakeVMIJson.SetP(namespace, "namespace")
	fakeVMIJson.SetP(configName, "spec.alertmanager.configMap")
	fakeVMIJson.SetP(templatesConfigName, "spec.alertmanager.templatesConfigMap")

	testServer, _, _ := getTestServerEnv(t, fakeVMIJson.String())

	c, err := newRestClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testclient.RestClient = c
	return &testclient
}
