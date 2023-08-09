FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/openresty:1.19.3.2-alpine

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk add --no-cache bash net-tools curl tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" >  /etc/timezone && apk del --no-cache tzdata
