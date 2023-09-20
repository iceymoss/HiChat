FROM golang:alpine3.17 AS builder
COPY /HiChat /HiChat

ENV GO111MODULE=on
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

WORKDIR /HiChat
RUN go mod download && \
    go build main.go

FROM ubuntu:20.04
COPY --from=builder /HiChat /HiChat

EXPOSE 8000

# ENTRYPOINT ["ls", "/HiChat"]
CMD ["/HiChat/main"]
