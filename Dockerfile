FROM ubuntu:20.04

RUN apt-get install -y ca-certificates

COPY build/linux/trigger /bin/trigger

RUN trigger --help

