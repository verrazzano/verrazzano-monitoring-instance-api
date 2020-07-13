// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	utiltesting "k8s.io/client-go/util/testing"
)

// used by unit tests to create a test server
func getTestServerEnv(t *testing.T, respBody string) (*httptest.Server, *utiltesting.FakeHandler, *metav1.Status) {

	status := &metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusSuccess}
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   http.StatusOK,
		ResponseBody: respBody,
		T:            t,
	}

	testServer := httptest.NewServer(&fakeHandler)
	return testServer, &fakeHandler, status
}

// Used by unit tests to create a test REST client
func newRestClient(testServer *httptest.Server) (*restclient.RESTClient, error) {
	c, err := restclient.RESTClientFor(&restclient.Config{
		Host: testServer.URL,
		ContentConfig: restclient.ContentConfig{
			GroupVersion:         &v1.SchemeGroupVersion,
			ContentType:          runtime.ContentTypeJSON,
			NegotiatedSerializer: serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs},
		},
		Username: "user",
		Password: "pass",
	})

	return c, err
}

// Generate a test configmap for the special case where the map has one key
// and its value is a string that is the content of a config file to expose.
func getTestConfigMap(configMapName string, namespace string, testMapPath string, testDataString string) *v1.ConfigMap {
	return getTestConfigMapFromMap(configMapName, namespace, map[string]string{
		testMapPath: testDataString,
	})
}

// Generate a test configmap from a string:string map
func getTestConfigMapFromMap(configMapName string, namespace string, data map[string]string) *v1.ConfigMap {
	c := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:       "12345678",
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: data,
	}

	return c
}

// Generate new instance of empty test config map.
func createEmptyTestConfigMap(configMapName string, namespace string) *v1.ConfigMap {

	c := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:       "12345678",
			Name:      configMapName,
			Namespace: namespace,
		},
	}

	return c
}

// Generate new instance of empty test secret.
func createEmptyTestSecret(secretName string, namespace string) *v1.Secret {

	s := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:       "12345678",
			Name:      secretName,
			Namespace: namespace,
		},
	}

	return s
}

// Generate k8s nodes with the specified external IPs set.
func createK8sTestNodes(externalIPs []string, privateWorker bool) (nodes *v1.NodeList) {

	nodes = &v1.NodeList{}
	for _, nextIP := range externalIPs {
		nextNode := v1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: nextIP, Labels: map[string]string{K8sPublicIpAddressLabel: nextIP}},
		}
		if !privateWorker {
			nextNode.Status = v1.NodeStatus{
				Addresses: []v1.NodeAddress{
					{Type: v1.NodeExternalIP, Address: nextIP},
				},
			}
		}
		nodes.Items = append(nodes.Items, nextNode)
	}

	return nodes
}
