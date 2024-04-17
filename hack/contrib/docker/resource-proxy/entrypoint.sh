#!/bin/sh
if [ "$1" = "bash" ];then
    exec /bin/bash
else
    exec /docker-entrypoint.sh "$@"
fi 