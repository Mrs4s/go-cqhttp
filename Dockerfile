#FROM golang:1.17-alpine AS builder
#RUN  sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
#
#RUN apk add --no-cache clang binutils clang-static musl-dev \
#  && go env -w CC=clang \
#  && go env -w CXX=clang++ \
#  && go env -w GO111MODULE=auto \
#  && go env -w CGO_ENABLED=1 \
#  && go env -w GOPROXY=https://goproxy.cn,direct
FROM golang:1.17-bullseye AS builder
COPY ./sources.list /etc/apt/sources.list
RUN apt-get update && \
    apt-get install -fy  build-essential clang \
    && go env -w CC=clang \
    && go env -w CXX=clang++ \
    && go env -w GO111MODULE=auto \
    && go env -w CGO_ENABLED=1 \
    && go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY ./ .

RUN set -ex \
    && cd /build \
    && go build -ldflags "-s -w -extldflags '-static'" -v -o cqhttp

FROM alpine:latest
RUN  sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache ffmpeg
COPY ./init.sh /
COPY --from=builder /build/cqhttp /usr/bin/cqhttp
RUN chmod +x /usr/bin/cqhttp && chmod +x /init.sh

WORKDIR /data

ENTRYPOINT [ "/init.sh" ]
