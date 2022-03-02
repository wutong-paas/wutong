#!/bin/bash
if [ "$1" = "bash" ];then
    exec /bin/bash
elif [ "$1" = "version" ];then
    /run/wutong-worker version
else
    exec /run/wutong-worker $@
fi