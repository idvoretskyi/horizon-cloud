FROM node:5
COPY horizon/ /horizon/

COPY build.sh /build.sh
RUN /build.sh

COPY run.sh /run.sh
CMD ["/run.sh"]
