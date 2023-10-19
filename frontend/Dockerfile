FROM nginx:1.23.1
ARG GIT_HASH=not_set
ARG STATIC_PATH=/usr/share/nginx/html/dist
RUN mkdir $STATIC_PATH
COPY ./build $STATIC_PATH
RUN echo "{\"commit\": \"$GIT_HASH\", \"microservice\": \"frontend\"}" > ${STATIC_PATH}/git.json
COPY ./frontend.conf /etc/nginx/conf.d/default.conf
