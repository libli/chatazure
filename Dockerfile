# builder
FROM golang:1.20-bullseye as builder

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download -x

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -X main.version=v1.1.0" -x -o chatazure .

# runner
FROM debian:bullseye-slim

# 安装ca-certificates
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

ENV TZ=Asia/Shanghai
# 设置环境变量
WORKDIR /app
COPY --from=builder /build/chatazure /app/chatazure
# 设置启动命令
CMD ["/app/chatazure"]