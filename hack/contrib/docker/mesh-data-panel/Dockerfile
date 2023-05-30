FROM  envoyproxy/envoy:v1.13.1
ARG RELEASE_DESC
LABEL "author"="dingpeng24001"
RUN apt-get update && apt-get install -y bash curl net-tools wget vim && \
    wget https://github.com/wutong-paas/files/raw/master/amd64/env2file -O /usr/bin/env2file    
ADD . /root/
RUN chmod 755 /root/start.sh && chmod 755 /usr/bin/env2file
ENV ENVOY_BINARY="/usr/local/bin/envoy"
ENV RELEASE_DESC=${RELEASE_DESC}
WORKDIR /root
ENTRYPOINT ["./start.sh"]