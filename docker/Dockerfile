# STEP 1 build executable binary

#FROM golang:alpine as builder
FROM golang:1.21.1-alpine3.18 as builder

RUN apk update
RUN apk add git

# Create appuser on builder image
RUN adduser -D -g '' appuser

COPY ./cmd/ /tmp/app/cmd
COPY go.* /tmp/app/
WORKDIR /tmp/app
RUN go mod tidy
RUN go env -w CGO_ENABLED=0
RUN go build -o /tmp/webhook github.com/udhos/k8s-mutating-admission-webhook/cmd/webhook

#
# STEP 2 build a small image from alpine
#
# curl: https://security.alpinelinux.org/srcpkg/curl
#
FROM alpine:3.18.0
COPY --from=builder /tmp/webhook /bin/webhook
#RUN apk add curl=8.1.2-r0 libcrypto3=3.1.0-r4 libssl3=3.1.0-r4
RUN apk update
RUN apk add curl
RUN apk upgrade libssl3 libcrypto3
RUN adduser -D -g '' user
USER user
ENTRYPOINT ["/bin/webhook"]
