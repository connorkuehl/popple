FROM golang:1.15-alpine AS build
RUN apk add build-base      # for gcc
RUN mkdir /popple
ADD . /popple
WORKDIR /popple
RUN go build .

FROM alpine:latest
WORKDIR /root/
COPY --from=build /popple/popple .

# docker image run --rm -v path/to/db:/root/popple.sqlite \
#                       -v path/to/token:/root/bot.token \
#                       image_name
ENTRYPOINT ["/root/popple"]
