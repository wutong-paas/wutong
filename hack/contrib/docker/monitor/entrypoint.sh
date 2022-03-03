#!/bin/ash
if [ "$1" = "bash" ];then
    exec /bin/ash
elif [ "$1" = "version" ];then
    /run/wutong-monitor version
else
    exec /run/wutong-monitor $@
fi