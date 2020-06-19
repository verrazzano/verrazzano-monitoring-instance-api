# API server for Verrazzano Monitoring Instance

[![build status](https://github.com/verrazzano/verrazzano-monitoring-instance-api/badges/master/build.svg)](https://github.com/verrazzano/verrazzano-monitoring-instance-api/commits/master)

## Overview

This document assumes basic familiarity with the
[verrazzano-monitoring-operator](https://github.com/verrazzano/verrazzano-monitoring-operator).

The API Server provides an API to configure certain aspects of a VMI (verrazzano-monitoring-instance), like Prometheus
rules and targets, and AlertManager integrations.

The API Server has no authentication or authorization features.  In VMO installations, calls to the API Server are proxied via
 `https` to `nginx` and basic auth is enabled.

## Building

```
make go-install
```

## Running Locally

After getting a VMO environment set up and creating an VMI, as in the [VMO usage doc](https://github.com/verrazzano/verrazzano-monitoring-operator/blob/master/docs/usage.md),
you can launch the API server locally, pointing to your Kubernetes cluster and the name of the VMI you've created: 

```
export KUBECONFIG=<your_kubeconfig>
make go-run VMI_NAME=<your_vmi>
```

Now, access http://localhost:9097 to explore the API Server Swagger.

## Running Tests

To run unit tests:

```
make unit-test
```

To run integration tests:

```
make integ-test
```

### Notes On Running Tests locally
* Unit tests require installing amtool & promtool in /opt/tools/bin  
   
## Regenerate the swagger docs
If you made any change to the API and need the swagger docs to be regenerated:
* Please make sure you have updated the documentation metadata at `cmd/cirith/main.go`  
* Run `make go-generate`  
* You will notice that `/static/cirith.json` is updated   
* Run `make go-run` to validate your changes  
* __Note__ : You would need to invalidate your browser cache or use a incognito tab to se the updates  
