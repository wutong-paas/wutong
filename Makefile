REGISTRY ?= swr.cn-southwest-2.myhuaweicloud.com/wutong
VERSION ?= v1.9.0

export REGISTRY
export VERSION

.PHONY: build
build-all:
	./multi-arch-build.sh all

build-api:
	./multi-arch-build.sh api

build-chaos:
	./multi-arch-build.sh chaos

build-gateway:
	./multi-arch-build.sh gateway

build-monitor:
	./multi-arch-build.sh monitor

build-mq:
	./multi-arch-build.sh mq

build-webcli:
	./multi-arch-build.sh webcli

build-worker:
	./multi-arch-build.sh worker

build-eventlog:
	./multi-arch-build.sh eventlog

build-init-probe:
	./multi-arch-build.sh init-probe

build-mesh-data-panel:
	./multi-arch-build.sh mesh-data-panel

build-wtctl:
	./multi-arch-build.sh wtctl

build-node:
	./multi-arch-build.sh node

build-resource-proxy:
	./multi-arch-build.sh resource-proxy
