ARG BASE_IMAGE_VERSION
FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC
RUN apk --no-cache add libstdc++ ca-certificates openssl openssl-dev

COPY build/libzmq/lib/libzmq.so.3 /usr/lib

COPY wutong-eventlog /run/wutong-eventlog
ADD entrypoint.sh /run/entrypoint.sh

EXPOSE 6366
EXPOSE 6365
EXPOSE 6364
EXPOSE 6363
EXPOSE 6362
EXPOSE 6607
VOLUME [ "/etc/wutongpaas" ]
ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT  ["/run/entrypoint.sh"]
