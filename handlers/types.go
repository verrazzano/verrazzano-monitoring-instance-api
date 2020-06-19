// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	k8sgo "k8s.io/client-go/kubernetes"
	restgo "k8s.io/client-go/rest"
)

// struct representing kubernetes rest client & clientset
type K8s struct {
	RestClient restgo.Interface
	ClientSet  k8sgo.Interface
	Config     *restgo.Config
}

func NewK8s(cfg *restgo.Config) (*K8s, error) {

	client := K8s{}

	// Initialize REST client
	s := schema.GroupVersion{Group: VMIGroup, Version: VMIVersion}
	cfg.GroupVersion = &s
	cfg.APIPath = "/apis"
	cfg.ContentType = runtime.ContentTypeJSON
	cfg.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(&runtime.Scheme{})}

	myRestClient, err := restgo.RESTClientFor(cfg)
	if err != nil {
		glog.Errorf("failure")
	}
	client.RestClient = myRestClient

	client.ClientSet, err = k8sgo.NewForConfig(cfg)
	if err != nil {
		glog.Errorf("failure")
	}
	// config is needed later when building SPDY executor; save in client
	client.Config = cfg
	return &client, nil
}
