FROM alpine:3.16

COPY ./build/linux/trigger /bin/trigger

RUN ls -al /bin
RUN trigger --help

