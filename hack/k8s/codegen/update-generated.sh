#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

vendor/k8s.io/code-generator/generate-groups.sh \
  "defaulter,client,lister,informer" \
  "github.com/wutong-paas/wutong/pkg/generated" \
  "github.com/wutong-paas/wutong/pkg/apis" \
  "wutong:v1alpha1" \
  --go-header-file "./hack/k8s/codegen/boilerplate.go.txt" \
  $@
