ARG BASE_IMAGE_VERSION
FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC
COPY . /run
RUN chmod +x /run/wutong-wtctl /run/entrypoint.sh
VOLUME [ "/rootfs/root","/rootfs/path","/ssl" ]
ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT ["/run/entrypoint.sh"]