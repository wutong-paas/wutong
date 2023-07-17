#!/bin/bash
set -e

if [ "$1" = "bash" ]; then
    exec /bin/bash
elif [ "$1" = "version" ]; then
    echo /run/wutong-mesh-data-panel version
elif [ "$1" = "run" ]; then
    /run/wutong-mesh-data-panel run || exit 1
else
    env2file conversion -f /run/envoy_config.yaml
    cluster_name=${POD_NAMESPACE}_${WT_PLUGIN_ID}_${WT_SERVICE_ALIAS}
    # start sidecar process
    /run/wutong-mesh-data-panel &
    # start envoy process
    exec envoy -c /run/envoy_config.yaml --service-cluster ${cluster_name} --service-node ${cluster_name}
fi
