FROM debian:8.4

RUN apt-get update && apt-get install -y ca-certificates

RUN yes '' | adduser --disabled-password hzc

ADD hzc-api /hzc-api
ADD run.sh /run.sh
ADD templates /templates

CMD ["/run.sh"]
