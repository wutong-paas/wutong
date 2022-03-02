#!/bin/bash
if [ "$1" = "bash" ];then
    exec /bin/bash
elif [ "$1" = "version" ];then
    /run/wutong-init-probe version
else
    exec /run/wutong-init-probe $@
fi