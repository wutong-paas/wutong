FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15

COPY wutong-mq /run/wutong-mq
ADD entrypoint.sh /run/entrypoint.sh
RUN chmod 655 /run/wutong-mq
EXPOSE 6300

ENTRYPOINT ["/run/entrypoint.sh"]

