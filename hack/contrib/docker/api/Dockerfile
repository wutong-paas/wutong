FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15
ARG RELEASE_DESC

ADD wutong-api /run/wutong-api
ADD entrypoint.sh /run/entrypoint.sh
WORKDIR /run
ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT ["/run/entrypoint.sh"]