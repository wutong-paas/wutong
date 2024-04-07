FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15

COPY . /run
RUN chmod +x /run/wutong-wtctl /run/entrypoint.sh
VOLUME [ "/rootfs/root","/rootfs/path","/ssl" ]

ENTRYPOINT ["/run/entrypoint.sh"]