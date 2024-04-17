FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15

ENV WORK_DIR=/run

RUN apk --no-cache add openssl openssh-client subversion
COPY wutong-chaos entrypoint.sh /run/
COPY export-app /src/export-app

WORKDIR $WORK_DIR

ENTRYPOINT ["/run/entrypoint.sh"]
