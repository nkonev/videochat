FROM nginx:1.23.1
ARG STATIC_PATH=/usr/share/nginx/html/dist
RUN mkdir -p $STATIC_PATH
COPY ./dist $STATIC_PATH
COPY ./frontend.conf /etc/nginx/nginx.conf
