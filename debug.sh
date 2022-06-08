#! /bin/sh 
dlv --headless --log --listen :9009 --api-version 2 --accept-multiclient debug cmd/api/main.go -- --kube-config=/Users/dp/.kube/config --start=true --enable-feature=privileged --mysql="root:de3d7b02@tcp(wt-db-rw.wt-system:3306)/region" --etcd=http://wt-etcd.wt-system:2379 --api-ssl-enable=true --api-addr-ssl=0.0.0.0:8443 --api-ssl-certfile=/Users/dp/temp/wtlocaldata/server.pem --api-ssl-keyfile=/Users/dp/temp/wtlocaldata/server.key.pem --client-ca-file=/Users/dp/temp/wtlocaldata/ca.pem --ws-addr=0.0.0.0:8080