FROM debian:8.4

RUN apt-get update && apt-get install -y ca-certificates

RUN yes '' | adduser --disabled-password hzc

ADD hzc-http /hzc-http
ADD run.sh /run.sh

CMD ["/run.sh"]
