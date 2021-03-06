// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package handlers

import (
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8sgo "k8s.io/client-go/kubernetes"
	restgo "k8s.io/client-go/rest"
)

// K8s struct representing kubernetes rest client & clientset
type K8s struct {
	RestClient restgo.Interface
	ClientSet  k8sgo.Interface
	Config     *restgo.Config
}

// NewK8s returns a new K8s struct
func NewK8s(cfg *restgo.Config) (*K8s, error) {
	client := K8s{}
	// Initialize REST client
	s := schema.GroupVersion{Group: VMIGroup, Version: VMIVersion}
	cfg.GroupVersion = &s
	cfg.APIPath = "/apis"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: serializer.NewCodecFactory(&runtime.Scheme{})}

	myRestClient, err := restgo.RESTClientFor(cfg)
	if err != nil {
		zap.S().Errorf("failure, Error: %s", err.Error())
	}
	client.RestClient = myRestClient

	client.ClientSet, err = k8sgo.NewForConfig(cfg)
	if err != nil {
		zap.S().Errorf("failure, Error: %s", err.Error())
	}
	// config is needed later when building SPDY executor; save in client
	client.Config = cfg
	return &client, nil
}
