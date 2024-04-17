FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/envoy:v1.16.5

LABEL "author"="dingpeng24001"
RUN apt-get update && apt-get install -y bash curl net-tools wget vim && \
    wget https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/amd/env2file -O /usr/bin/env2file    
ADD . /run/

RUN chmod 755 /run/entrypoint.sh && chmod 755 /usr/bin/env2file
ENV ENVOY_BINARY="/usr/local/bin/envoy"

WORKDIR /run
ENTRYPOINT ["/run/entrypoint.sh"]