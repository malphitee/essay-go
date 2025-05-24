# 多阶段构建Dockerfile

# 阶段1: 构建Go应用 (原后端构建阶段)
FROM golang:1.24.2-alpine AS builder
# 设置Go模块代理为国内镜像
ENV GOPROXY=https://goproxy.cn,direct
# 先配置Alpine的国内镜像源，再安装git
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache git

WORKDIR /app
# 复制go.mod和go.sum
COPY go.mod go.sum ./
# 下载依赖
# RUN go mod download
# 使用 tidy -e 来处理潜在的错误，并确保所有依赖都被记录
RUN go mod tidy -e

# 复制所有项目文件 (包括 .go 文件和 templates 目录等)
# 确保 templates 目录在构建上下文中
COPY . .

# 静态编译Go应用
# 输出文件名可以保持为 essay-api 或改为 app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o /app/essay-server .

# 阶段2: 最终运行镜像
FROM alpine:3.19
# 使用国内镜像
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

WORKDIR /app

# 从构建阶段复制编译好的Go应用二进制文件
COPY --from=builder /app/essay-server /app/essay-server
# 复制templates目录
# 确保 main.go 中的 router.LoadHTMLGlob("templates/*") 能找到这个路径
COPY --from=builder /app/templates ./templates

# 暴露应用端口
EXPOSE 8080

# 设置环境变量
ENV GIN_MODE=release

# 定义DeepSeek API相关环境变量
ENV DEEPSEEK_API_KEY=""
ENV DEEPSEEK_MODEL="deepseek-chat"

# 启动应用
CMD ["/app/essay-server"]
