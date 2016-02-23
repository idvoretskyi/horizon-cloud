FROM alpine:3.3

RUN apk update && \
    apk add bash openssh rsync && \
    rm -rf /var/cache/apk/*

EXPOSE 22

COPY run.sh /run.sh

CMD ["/run.sh"]
