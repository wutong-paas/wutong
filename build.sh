#! /bin/bash

export VERSION=v1.4.0-alpha1
./release-new.sh all push


# export NAMESPACE=wt-api
# export VERSION=v1.4.0-test
# docker buildx create --use --name wutongbuilder || docker buildx use wutongbuilder
# # docker buildx build --platform linux/amd64,linux/arm64 --push -t swr.cn-southwest-2.myhuaweicloud.com/wutong/${NAMESPACE}:${VERSION} -f Dockerfile.multiarch . 
# docker buildx build --platform linux/amd64,linux/arm64 --push -t poneding/${NAMESPACE}:${VERSION} -f Dockerfile.api . 
# # docker buildx rm wutongbuilder