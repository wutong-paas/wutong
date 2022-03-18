ARG BASE_IMAGE_VERSION
ARG VERSION
FROM --platform=${TARGETPLATFORM} swr.cn-southwest-2.myhuaweicloud.com/wutong/wt-all:${VERSION} AS wt-all

FROM --platform=${TARGETPLATFORM} wutongpaas/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC
ARG TARGETOS TARGETARCH TARGETPLATFORM
RUN apk --no-cache add libstdc++ ca-certificates openssl openssl-dev

COPY build/libzmq/lib/libzmq.so.3 /usr/lib

COPY --from=wt-all /output/bin/wutong-eventlog /run/wutong-eventlog
ADD entrypoint.sh /run/entrypoint.sh

EXPOSE 6366
EXPOSE 6365
EXPOSE 6364
EXPOSE 6363
EXPOSE 6362
VOLUME [ "/etc/wutongpaas" ]
ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT  ["/run/entrypoint.sh"]
