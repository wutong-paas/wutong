ARG BASE_IMAGE_VERSION
FROM wutongpaas/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC

ADD wutong-api /run/wutong-api
ADD entrypoint.sh /run/entrypoint.sh
WORKDIR /run
ENV RELEASE_DESC=${RELEASE_DESC}
VOLUME [ "/etc/wutongpaas" ]
ENTRYPOINT ["/run/entrypoint.sh"]