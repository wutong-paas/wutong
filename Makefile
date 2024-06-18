REGISTRY ?= swr.cn-southwest-2.myhuaweicloud.com/wutong
VERSION ?= v1.13.0

export REGISTRY
export VERSION

.PHONY: build
build-all:
	./build.sh all

build-api:
	./build.sh api

build-chaos:
	./build.sh chaos

build-gateway:
	./build.sh gateway

build-monitor:
	./build.sh monitor

build-mq:
	./build.sh mq

build-webcli:
	./build.sh webcli

build-worker:
	./build.sh worker

build-eventlog:
	./build.sh eventlog

build-init-probe:
	./build.sh init-probe

build-mesh-data-panel:
	./build.sh mesh-data-panel

build-wtctl:
	./build.sh wtctl

build-node:
	./build.sh node

build-resource-proxy:
	./build.sh resource-proxy
