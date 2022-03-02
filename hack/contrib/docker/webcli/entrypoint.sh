#!/bin/bash
if [ "$1" = "bash" ];then
    exec /bin/bash
elif [ "$1" = "version" ];then
    /usr/bin/wutong-webcli version
else
    exec /usr/bin/wutong-webcli $@
fi