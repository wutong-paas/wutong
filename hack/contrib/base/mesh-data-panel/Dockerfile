FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/envoy:v1.16.5

RUN apt-get update && apt-get install -y bash curl net-tools wget vim && \
    if [ $(uname -m) = "aarch64" ]; then \
        wget https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/arm/env2file -O /usr/bin/env2file; \
    else \
        wget https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/amd/env2file -O /usr/bin/env2file; \
    fi && chmod 755 /usr/bin/env2file

ENV ENVOY_BINARY="/usr/local/bin/envoy"