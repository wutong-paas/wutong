FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15

COPY wutong-worker /run/wutong-worker
COPY entrypoint.sh /run/entrypoint.sh
RUN chmod 655 /run/wutong-worker

ENV EX_DOMAIN=ali-sh.wutong.net:10080

ENTRYPOINT ["/run/entrypoint.sh"]
