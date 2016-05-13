FROM node:5

RUN npm install -g horizon@0.3.2-1
RUN yes '' | adduser --disabled-password horizon
RUN su -s /bin/sh horizon -c 'cd ~ && hz init app'

COPY run.sh /run.sh
CMD ["/run.sh"]
