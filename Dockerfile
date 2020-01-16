############################
# Build container
############################
FROM golang:1.13-stretch AS dep

WORKDIR /go

ADD vault.go .
ADD vendor/ /go/src/

RUN go build vault.go

############################
# Final container
############################
FROM registry.cto.ai/official_images/base:latest

WORKDIR /ops

RUN apt-get update && apt-get install ca-certificates -y
COPY --from=dep /go/vault /usr/local/bin/vault
