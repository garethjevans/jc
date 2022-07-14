FROM alpine:3.14

COPY ./build/linux/trigger /bin/trigger

RUN ls /bin
RUN trigger --help

