module github.com/verrazzano/verrazzano-monitoring-instance-api

go 1.13

require (
	github.com/Jeffail/gabs/v2 v2.2.0
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190911111923-ecfe977594f1 // indirect
	github.com/go-swagger/go-swagger v0.21.0
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/json-iterator/go v1.1.7 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.15.7
	k8s.io/apimachinery v0.15.7
	k8s.io/client-go v0.15.7
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// pinning kubernetes-1.15.7 - latest in 1.15.* series, which was first to use go.mod
	k8s.io/api => k8s.io/api v0.15.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.15.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.15.7
	k8s.io/client-go => k8s.io/client-go v0.15.7
	k8s.io/code-generator => k8s.io/code-generator v0.15.7
)
