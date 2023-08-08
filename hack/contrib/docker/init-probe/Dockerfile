FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/alpine:3.15
ARG RELEASE_DESC
COPY . /run/
RUN chmod 655 /run/wutong-init-probe /run/entrypoint.sh
ENV RELEASE_DESC=${RELEASE_DESC}
ENTRYPOINT [ "/run/entrypoint.sh" ]
CMD ["probe"]

