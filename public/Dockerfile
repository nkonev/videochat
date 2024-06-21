FROM node:22.1.0-alpine
ARG DEST=/opt/public
RUN mkdir -p $DEST
WORKDIR $DEST
ENV NODE_ENV=production
COPY ./package.json $DEST
COPY ./package-lock.json $DEST
COPY ./node_modules $DEST/node_modules
COPY ./server $DEST/server
COPY ./common $DEST/common
COPY ./dist $DEST/dist
ENTRYPOINT ["node", "./server"]
