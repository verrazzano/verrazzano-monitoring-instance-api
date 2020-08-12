# API server for Verrazzano Monitoring Instance


## Overview

This document assumes basic familiarity with the
[verrazzano-monitoring-operator](https://github.com/verrazzano/verrazzano-monitoring-operator).

The API Server provides an API to configure certain aspects of a VMI (verrazzano-monitoring-instance), like Prometheus
rules and targets, and AlertManager integrations.

The API Server has no built-in authentication or authorization features.  In Verrazzano installations, calls to the
API Server are proxied via `https` to `nginx` and basic authentication is enforced there.

## Building

To build the API server:

```
make go-install
```

## Running Locally

Set up a Verrazzano environment.  This will include the Verrazzano Monitoring Instance, as described
in the [Verrazzano Monitoring Operator usage documentation](https://github.com/verrazzano/verrazzano-monitoring-operator/blob/master/docs/usage.md).

Launch the API server locally, pointing to your Kubernetes cluster and the name of the Verrazzano Monitoring Instance:

```
export KUBECONFIG=<your_kubeconfig>
make go-run VMI_NAME=<your_vmi>
```

The API Server will be available on [http://localhost:9097](http://localhost:9097).

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
* Unit tests require installing promtool in /opt/tools/bin  
   
## Contributing to Verrazzano

Oracle welcomes contributions to this project from anyone.  Contributions may be reporting an issue with the operator or submitting a pull request.  Before embarking on significant development that may result in a large pull request, it is recommended that you create an issue and discuss the proposed changes with the existing developers first.

If you want to submit a pull request to fix a bug or enhance an existing feature, please first open an issue and link to that issue when you submit your pull request.

If you have any questions about a possible submission, feel free to open an issue too.

## Contributing to the Verrazzano Monitoring Instance API Server repository

Pull requests can be made under The Oracle Contributor Agreement (OCA), which is available at [https://www.oracle.com/technetwork/community/oca-486395.html](https://www.oracle.com/technetwork/community/oca-486395.html).

For pull requests to be accepted, the bottom of the commit message must have the following line, using the contributor’s name and e-mail address as it appears in the OCA Signatories list.

```
Signed-off-by: Your Name <you@example.org>
```

This can be automatically added to pull requests by committing with:

```
git commit --signoff
```

Only pull requests from committers that can be verified as having signed the OCA can be accepted.

## Pull request process

*       Fork the repository.
*       Create a branch in your fork to implement the changes. We recommend using the issue number as part of your branch name, for example, `1234-fixes`.
*       Ensure that any documentation is updated with the changes that are required by your fix.
*       Ensure that any samples are updated if the base image has been changed.
*       Submit the pull request. Do not leave the pull request blank. Explain exactly what your changes are meant to do and provide simple steps on how to validate your changes. Ensure that you reference the issue you created as well. We will assign the pull request to 2-3 people for review before it is merged.

## Introducing a new dependency

Please be aware that pull requests that seek to introduce a new dependency will be subject to additional review.  In general, contributors should avoid dependencies with incompatible licenses, and should try to use recent versions of dependencies.  Standard security vulnerability checklists will be consulted before accepting a new dependency.  Dependencies on closed-source code, including WebLogic Server, will most likely be rejected.
