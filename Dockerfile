FROM swr.cn-southwest-2.myhuaweicloud.com/wutong/golang:1.20

COPY ./app app

RUN chmod 777 ./app

CMD ./app
