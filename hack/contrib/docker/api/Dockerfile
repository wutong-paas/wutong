FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/wt-api-base:stable

ADD wutong-api /run/wutong-api
ADD entrypoint.sh /run/entrypoint.sh
WORKDIR /run

ENTRYPOINT ["/run/entrypoint.sh"]