FROM ubuntu:20.04

RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

COPY build/linux/trigger /bin/trigger

RUN trigger --help

