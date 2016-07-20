FROM node:5

COPY hzc-web-sync/package.json /hzc-web-sync/
RUN cd /hzc-web-sync && npm install

COPY hzc-web-sync /hzc-web-sync
RUN cd /hzc-web-sync && npm run build

COPY run.sh /run.sh
CMD ["/run.sh"]
