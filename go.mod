// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

module github.com/verrazzano/verrazzano-monitoring-instance-api

go 1.13

replace (
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
)
require (
	github.com/Jeffail/gabs/v2 v2.2.0
	github.com/go-swagger/go-swagger v0.21.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f // indirect
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v0.18.2
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/yaml v1.2.0
)
