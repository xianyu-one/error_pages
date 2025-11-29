# Build Stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
# 禁用 CGO 以构建静态二进制文件
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o error_pages .

# Run Stage
FROM scratch
WORKDIR /
COPY --from=builder /app/error_pages .
EXPOSE 80
ENTRYPOINT ["/error_pages"]