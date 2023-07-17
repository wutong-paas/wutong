ARG BASE_IMAGE_VERSION
FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:${BASE_IMAGE_VERSION}
ARG RELEASE_DESC

ADD wutong-webcli /run/
ADD entrypoint.sh /run/entrypoint.sh
RUN mkdir /root/.kube

ENV PORT 7070

ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT ["/run/entrypoint.sh"]