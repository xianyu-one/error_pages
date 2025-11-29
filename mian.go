package main

import (
	_ "embed"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed index.html
var rawHTML string

// 预渲染缓存：Key为状态码字符串 (e.g., "404"), Value为完整的HTML字节
var pageCache map[string][]byte

// 定义需要预渲染的常用状态码列表
// 涵盖了 Caddy 可能产生的大部分错误码
var commonCodes = []string{
	"400", "401", "403", "404", "405", "408", "429",
	"500", "501", "502", "503", "504",
	// 一些非标准或Caddy特定的
	"520", "521", "522", "523",
}

// 占位符定义 (与你的 HTML 对应)
const (
	placeholderStatus = `{{placeholder "http.error.status_code"}}`
	placeholderReqID  = `{{placeholder "http.request.header.X-Request-ID"}}`
)

func main() {
	port := flag.String("port", "80", "Server port")
	flag.Parse()

	// 1. 初始化缓存
	initCache()

	// 2. 设置路由
	http.HandleFunc("/", handler)

	// 3. 启动服务
	addr := ":" + *port
	log.Printf("Error Pages Service started on %s (PID: %d)\n", addr, os.Getpid())
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// initCache 在启动时将常用状态码的 HTML 渲染好放入内存
func initCache() {
	pageCache = make(map[string][]byte, len(commonCodes))

	// 预先清理掉 Request ID 的占位符，因为在静态缓存模式下无法动态注入每个请求的 ID
	// 如果不清理，页面上会显示 ugly 的占位符字符串
	// 这里将其替换为空字符串，前端 JS 会优雅降级
	baseHTML := strings.ReplaceAll(rawHTML, placeholderReqID, "")

	for _, code := range commonCodes {
		// 替换状态码
		rendered := strings.ReplaceAll(baseHTML, placeholderStatus, code)
		pageCache[code] = []byte(rendered)
	}
	log.Printf("Pre-rendered %d status pages into memory.\n", len(pageCache))
}

// handler 处理请求
func handler(w http.ResponseWriter, r *http.Request) {
	// 获取 URL Path，例如访问 /404 -> 获取 "404"
	// TrimPrefix 去掉开头的 "/"
	path := strings.TrimPrefix(r.URL.Path, "/")

	// 如果路径为空（直接访问根目录），默认显示 404 或自定义首页
	if path == "" {
		path = "404"
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 可以在这里加一个 Cache-Control，让浏览器短时间缓存
	// w.Header().Set("Cache-Control", "public, max-age=60")

	// 1. 快速路径：命中预渲染缓存
	if data, ok := pageCache[path]; ok {
		w.WriteHeader(http.StatusOK) // 注意：这里返回 200，因为这是错误页面的“内容”，Caddy 会在外部 replace_status
		_, _ = w.Write(data)
		return
	}

	// 2. 慢速路径：未知的状态码（例如 /418）
	// 实时进行字符串替换
	// 为了保持高性能，我们基于已经去除了 RequestID 的模板进行替换
	// 注意：这里需要再次清理 RequestID，或者我们复用一个已经清理过的 base 模板
	// 简单起见，这里直接操作 rawHTML，虽然稍微慢一点点，但很少触发
	out := strings.ReplaceAll(rawHTML, placeholderStatus, path)
	out = strings.ReplaceAll(out, placeholderReqID, "")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}
