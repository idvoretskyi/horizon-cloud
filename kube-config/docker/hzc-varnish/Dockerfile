FROM ubuntu:15.10

RUN apt-get -qq update && \
    apt-get install -y varnish && \
    rm -f /etc/varnish/default.vcl

COPY default.vcl.sh /etc/varnish/default.vcl.sh
COPY run.sh /run.sh

CMD ["/run.sh"]
