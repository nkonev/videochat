FROM nginx:1.16.0
ARG STATIC_PATH=/usr/share/nginx/html/build
RUN mkdir $STATIC_PATH
COPY ./build $STATIC_PATH
COPY ./frontend.conf /etc/nginx/conf.d/default.conf