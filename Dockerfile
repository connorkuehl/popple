FROM golang:1.17.0-alpine AS build
RUN apk add build-base git
RUN mkdir /popple
ADD . /popple
WORKDIR /popple
RUN make

FROM alpine:latest
WORKDIR /root/

COPY --from=build /popple/popple .

# docker run --rm -v path/to/db:/root/popple.sqlite \
#                       -v path/to/token:/root/bot.token \
#                       image_name
ENTRYPOINT ["/root/popple"]
