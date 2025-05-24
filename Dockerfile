# 多阶段构建Dockerfile

# 阶段1: 构建Go应用 (原后端构建阶段)
FROM golang:1.24.2-alpine AS builder

# 设置Go模块代理为国内镜像和构建参数
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0 
ENV GOOS=linux 
ENV GOARCH=amd64

# 先配置Alpine的国内镜像源，再安装git
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# 创建非 root 用户
RUN adduser -D -g '' appuser

WORKDIR /app

# 复制go.mod和go.sum并下载依赖
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 复制源代码
COPY . .

# 使用缓存和并行构建加速编译
RUN go build -ldflags="-s -w" -o /app/essay-server .

# 阶段2: 最终运行镜像
FROM alpine:3.19

# 使用国内镜像并安装必要的包
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

# 创建非 root 用户
RUN adduser -D -g '' appuser

# 创建应用目录并设置权限
WORKDIR /app
RUN mkdir -p /app/data

# 从构建阶段复制编译好的二进制文件和必要文件
COPY --from=builder /app/essay-server /app/
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/data /app/data

# 设置正确的文件权限
RUN chmod +x /app/essay-server && \
    chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露应用端口
EXPOSE 8080

# 设置环境变量
ENV GIN_MODE=release

# 定义DeepSeek API相关环境变量
ENV DEEPSEEK_API_KEY=""
ENV DEEPSEEK_MODEL="deepseek-chat"

# 定义AWS相关环境变量
ENV AWS_ACCESS_KEY_ID=""
ENV AWS_SECRET_ACCESS_KEY=""
ENV AWS_REGION="ap-northeast-1"
ENV DYNAMODB_TABLE="essay"
ENV ENABLE_DYNAMODB="false"

ENV JWT_SECRET="your_secret_key"

# 启动应用
CMD ["/app/essay-server"]
