FROM node:5

COPY run.sh /run.sh
COPY rethinkdb/build/release/rethinkdb /rethinkdb

RUN yes '' | adduser --disabled-password rethinkdb

CMD ["/run.sh"]
