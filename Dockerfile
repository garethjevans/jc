FROM ubuntu:20.04

COPY build/linux/trigger /bin/trigger

RUN ls -al /bin
RUN trigger --help

