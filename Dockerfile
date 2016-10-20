FROM alpine:latest
MAINTAINER weidewang dewang.wei@gmail.com

ENV  TZ="Asia/Shanghai"
RUN apk add --update --no-cache ca-certificates tzdata curl

cp objs/reverse /opt/reverse
WORKDIR /opt/
EXPOSE 6000
ENTRYPOINT ["/opt/reverse"]
