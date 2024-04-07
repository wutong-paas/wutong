FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/prometheus:v2.20.0

USER root
VOLUME ["/prometheusdata"]

COPY wutong-monitor /run/wutong-monitor

ADD entrypoint.sh /run/entrypoint.sh

ENTRYPOINT ["/run/entrypoint.sh"]