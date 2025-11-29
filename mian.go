package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed index.html
var rawHTML string

var pageCache map[string][]byte

var commonCodes = []string{
	"400", "401", "403", "404", "405", "408", "429",
	"500", "501", "502", "503", "504",
	"520", "521", "522", "523",
}

const (
	placeholderStatus = `{{placeholder "http.error.status_code"}}`
	placeholderReqID  = `{{placeholder "http.request.header.X-Request-ID"}}`
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80" // 如果没有设置环境变量，默认为 80
	}

	initCache()

	http.HandleFunc("/", handler)

	addr := ":" + port
	log.Printf("Error Pages Service started on %s (PID: %d)\n", addr, os.Getpid())
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// initCache 在启动时将常用状态码的 HTML 渲染好放入内存
func initCache() {
	pageCache = make(map[string][]byte, len(commonCodes))
	baseHTML := strings.ReplaceAll(rawHTML, placeholderReqID, "")
	for _, code := range commonCodes {
		rendered := strings.ReplaceAll(baseHTML, placeholderStatus, code)
		pageCache[code] = []byte(rendered)
	}
	log.Printf("Pre-rendered %d status pages into memory.\n", len(pageCache))
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "404"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if data, ok := pageCache[path]; ok {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	out := strings.ReplaceAll(rawHTML, placeholderStatus, path)
	out = strings.ReplaceAll(out, placeholderReqID, "")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}
