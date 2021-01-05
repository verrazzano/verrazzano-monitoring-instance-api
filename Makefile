# Copyright (C) 2020, 2021, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

NAME:=verrazzano-monitoring-instance-api

DOCKER_IMAGE_NAME ?= ${NAME}-dev
DOCKER_IMAGE_TAG ?= local-$(shell git rev-parse --short HEAD)

ifeq ($(MAKECMDGOALS),$(filter $(MAKECMDGOALS),push push-tag))
ifndef DOCKER_REPO
    $(error DOCKER_REPO must be defined as the name of the docker repository where image will be pushed)
endif
ifndef DOCKER_NAMESPACE
    $(error DOCKER_NAMESPACE must be defined as the name of the docker namespace where image will be pushed)
endif
    DOCKER_IMAGE_FULLNAME = ${DOCKER_REPO}/${DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}
endif

DIST_DIR:=dist
BIN_DIR:=${DIST_DIR}/bin
BIN_NAME:=${NAME}
GO ?= go
VMI_NAME ?= vmi-1
VMI_NAMESPACE ?= default
VMI_DEBUG_LEVEL ?= 2

.PHONY: all
all: build

#
# Go build related tasks
#

.PHONY: go-install
go-install: go-generate
	export GO111MODULE=on; $(GO) install ./cmd/cirith/...


.PHONY: go-run
go-run: go-install
	$(GO) run ./cmd/cirith/* --kubeconfig=${KUBECONFIG} --v=4 --vminame ${VMI_NAME} --staticPath=./static


.PHONY: go-clean
go-clean:
	rm -rf ${DIST_DIR}

.PHONY: go-fmt
go-fmt:
	gofmt -s -e -d $(shell find . -name "*.go" | grep -v /vendor/) > error.txt
	if [ -s error.txt ]; then\
		cat error.txt;\
		rm error.txt;\
		exit 1;\
	fi
	rm error.txt

.PHONY: go-vet
go-vet:
	$(GO) vet $(shell go list ./... | grep -v /vendor/)

.PHONY: go-lint
go-lint:
	$(GO) get -u golang.org/x/lint/golint
	golint -set_exit_status $(shell go list ./... | grep -v /vendor/)

.PHONY: go-ineffassign
go-ineffassign:
	$(GO) get -u github.com/gordonklaus/ineffassign
	ineffassign $(shell go list ./...)

#
# Docker-related tasks
#
.PHONY: docker-clean
docker-clean:
	rm -rf ${DIST_DIR}

.PHONY: build
build:
	echo ${VERSION} ${GITLAB_CI} ${CI_COMMIT_TAG} ${CI_COMMIT_SHA}

	mkdir -p ${BIN_DIR}
	cp -r static ${DIST_DIR}/bin
	docker build \
		--pull --no-cache \
		-t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: push
push: build
	docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	docker push ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}

	if [ "${CREATE_LATEST_TAG}" == "1" ]; then \
		docker tag ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:latest; \
		docker push ${DOCKER_IMAGE_FULLNAME}:latest; \
	fi


.PHONY: push-tag
push-tag:
	docker pull ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG}
	docker tag ${DOCKER_IMAGE_FULLNAME}:${DOCKER_IMAGE_TAG} ${DOCKER_IMAGE_FULLNAME}:${TAG_NAME}
	docker push ${DOCKER_IMAGE_FULLNAME}:${TAG_NAME}


.PHONY: unit-test
unit-test:
	export GO111MODULE=on; $(GO) test -v ./handlers/...

