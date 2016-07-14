FROM node:5

# This is written this way so that it's extremely quick to build and
# deploy during development if you aren't changing any dependencies.

COPY setup.sh /
RUN /setup.sh

COPY hzc-web-backend/package.json /hzc-web-backend/
RUN cd /hzc-web-backend && npm install

COPY hzc-web-backend /hzc-web-backend
RUN cd /hzc-web-backend && npm run build

COPY run.sh /run.sh
CMD ["/run.sh"]
