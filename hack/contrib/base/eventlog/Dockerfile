FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15

RUN if [ $(uname -m) = "aarch64" ]; then \
        apk add --no-cache libstdc++ ca-certificates openssl openssl-dev zeromq-dev; \
    else \
        apk add --no-cache libstdc++ ca-certificates openssl openssl-dev; \
    fi
