FROM alpine:3.14

COPY ./build/linux/trigger /bin/trigger

ENTRYPOINT ["/bin/trigger"]

RUN ls /bin
RUN /bin/trigger --help

