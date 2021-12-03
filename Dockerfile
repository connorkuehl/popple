FROM golang:1.17.0-alpine AS build
RUN mkdir /builddir
ADD . /builddir
WORKDIR /builddir
RUN go build ./...

FROM alpine:latest
RUN mkdir /data
WORKDIR /data
COPY --from=build /builddir/popple /usr/local/bin/popple

ENTRYPOINT ["/usr/local/bin/popple"]
