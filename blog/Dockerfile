FROM node:22.1.0-alpine
RUN mkdir /opt/blog
WORKDIR /opt/blog
ENV NODE_ENV=production
COPY ./package.json /opt/blog
COPY ./package-lock.json /opt/blog
COPY ./node_modules /opt/blog/node_modules
COPY ./server /opt/blog/server
COPY ./common /opt/blog/common
COPY ./dist /opt/blog/dist
ENTRYPOINT ["node", "./server"]
