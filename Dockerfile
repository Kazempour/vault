############################
# Build container
############################
FROM golang:1.13-alpine AS dep

WORKDIR /ops

ADD . .

RUN go build vault.go

############################
# Final container
############################
FROM alpine:latest

WORKDIR /ops

COPY --from=dep /ops/vault /usr/local/bin/vault
