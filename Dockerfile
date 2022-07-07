FROM golang:1.18-alpine AS builder

RUN go env -w GO111MODULE=auto \
  && go env -w CGO_ENABLED=0 \
  && go env -w GOPROXY=https://goproxy.cn,direct 

WORKDIR /build

COPY ./ .

RUN set -ex \
    && cd /build \
    && go build -ldflags "-s -w -extldflags '-static'" -o cqhttp

FROM alpine:latest

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache ffmpeg

COPY --from=builder /build/cqhttp /usr/bin/cqhttp
RUN chmod +x /usr/bin/cqhttp

WORKDIR /data

ENTRYPOINT [ "/usr/bin/cqhttp" ]
