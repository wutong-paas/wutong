#!/bin/bash
if [ "$1" = "bash" ];then
    exec /bin/bash
elif [ "$1" = "version" ];then
    /run/wutong-wtctl version
elif [ "$1" = "copy" ];then
    cp -a /run/wutong-wtctl /rootfs/usr/local/bin/
else
    exec /run/wutong-wtctl "$@"
fi