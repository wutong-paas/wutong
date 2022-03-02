#!/bin/bash
if [ "$1" = "bash" ];then
    exec /bin/bash
elif [ "$1" = "version" ];then
    /run/wutong-eventlog version
else
    exec /run/wutong-eventlog $@
fi