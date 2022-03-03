ARG BASE_IMAGE_VERSION
FROM wutongpaas/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC
ENV PORT 7070

ADD wutong-webcli /usr/bin/wutong-webcli
ADD entrypoint.sh /entrypoint.sh
RUN mkdir /root/.kube

ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT ["/entrypoint.sh"]