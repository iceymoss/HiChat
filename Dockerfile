FROM golang:1.17-alpine AS builder

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .
# 复制配置文件
COPY --from=builder /app/config-debug.yaml .
# 复制 index.html（根目录的首页文件）
COPY --from=builder /app/index.html .
# 复制 views 目录（HTML 模板）
COPY --from=builder /app/views ./views
# 复制 asset 目录（静态资源）
COPY --from=builder /app/asset ./asset

EXPOSE 8000

CMD ["./main"]