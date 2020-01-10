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

COPY --from=dep /go/vault /usr/local/bin/vault
