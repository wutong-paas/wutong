FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15

ADD wutong-webcli /run/
ADD entrypoint.sh /run/entrypoint.sh
RUN mkdir /root/.kube

ENV PORT 7070

ENTRYPOINT ["/run/entrypoint.sh"]