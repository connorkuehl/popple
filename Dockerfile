FROM golang:1.15-alpine AS build
RUN apk add build-base git
RUN mkdir /popple
ADD . /popple
WORKDIR /popple
RUN make build

FROM alpine:latest
RUN apk add curl
WORKDIR /root/
COPY --from=build /popple/popple .

# docker image run --rm -v path/to/db:/root/popple.sqlite \
#                       -v path/to/token:/root/bot.token \
#                       image_name
ENTRYPOINT ["/root/popple"]

# TODO: parameterize the port. 8080 is Popple's default
HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost:8080/healthy
