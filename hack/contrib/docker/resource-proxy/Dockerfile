FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/nginx:1.19
ARG RELEASE_DESC

COPY entrypoint.sh /run/entrypoint.sh
COPY resource-proxy.conf /etc/nginx/conf.d/

VOLUME ["/data/nginx/cache"]
ENV RELEASE_DESC=${RELEASE_DESC}

RUN chmod +x /run/entrypoint.sh
ENTRYPOINT [ "/run/entrypoint.sh" ]
CMD ["nginx", "-g", "daemon off;"]