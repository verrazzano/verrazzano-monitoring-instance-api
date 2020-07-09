// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano-monitoring-instance-api

go 1.13

require (
	github.com/Jeffail/gabs/v2 v2.2.0
	github.com/go-swagger/go-swagger v0.21.0
	github.com/gorilla/mux v1.7.3
	github.com/rs/zerolog v1.20.0
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.16.0
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/yaml v1.2.0
)
