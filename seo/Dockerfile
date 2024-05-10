FROM node:22.1.0-alpine
RUN mkdir /opt/seo
WORKDIR /opt/seo
ENV NODE_ENV=production
COPY ./package.json /opt/seo
COPY ./package-lock.json /opt/seo
COPY ./node_modules /opt/seo/node_modules
COPY ./server /opt/seo/server
COPY ./common /opt/seo/common
COPY ./dist /opt/seo/dist
ENTRYPOINT ["node", "./server"]
