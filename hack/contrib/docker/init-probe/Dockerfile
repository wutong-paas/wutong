FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15

COPY . /run/
RUN chmod 655 /run/wutong-init-probe /run/entrypoint.sh

ENTRYPOINT [ "/run/entrypoint.sh" ]
CMD ["probe"]

