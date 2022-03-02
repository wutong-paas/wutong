ARG BASE_IMAGE_VERSION
FROM wutongpaas/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC
COPY wutong-mq /run/wutong-mq
ADD entrypoint.sh /run/entrypoint.sh
RUN chmod 655 /run/wutong-mq
EXPOSE 6300

ENV RELEASE_DESC=${RELEASE_DESC}

ENTRYPOINT ["/run/entrypoint.sh"]

