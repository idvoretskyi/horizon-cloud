FROM ubuntu:16.04

RUN apt-get update && \
    apt-get install -y stunnel

COPY stunnel.conf.bash /stunnel.conf.bash
COPY run.sh /run.sh

CMD ["/run.sh"]
