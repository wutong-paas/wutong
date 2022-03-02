ARG BASE_IMAGE_VERSION
FROM wutongpaas/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC
COPY . /run/
RUN chmod 655 /run/wutong-init-probe /run/entrypoint.sh
ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT [ "/run/entrypoint.sh" ]
CMD ["probe"]

