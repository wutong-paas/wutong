ARG BASE_IMAGE_VERSION
FROM wutongpaas/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC
COPY . /run
RUN chmod +x /run/wutong-grctl /run/entrypoint.sh
VOLUME [ "/rootfs/root","/rootfs/path","/ssl" ]
ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT ["/run/entrypoint.sh"]