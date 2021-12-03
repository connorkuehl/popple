FROM golang:1.17.0-alpine AS build
RUN apk add build-base git
RUN mkdir /builddir
ADD . /builddir
WORKDIR /builddir
RUN make

FROM alpine:latest
RUN mkdir /data
WORKDIR /data
COPY --from=build /builddir/popple /usr/local/bin/popple

ENTRYPOINT ["/usr/local/bin/popple"]
