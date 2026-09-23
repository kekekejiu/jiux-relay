package main

import (
	"embed"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed templates/*.html
var tplFS embed.FS

//go:embed static/*
var staticFS embed.FS

var tpl *template.Template

func main() {
	addr := flag.String("addr", envOr("RELAY_ADDR", "127.0.0.1:8080"), "监听地址")
	dbPath := flag.String("db", envOr("RELAY_DB", "app.db"), "SQLite 文件路径")
	adminPath := flag.String("admin", envOr("RELAY_ADMIN_PATH", "/manage"), "后台路径(建议改成隐蔽路径)")
	flag.Parse()

	abs, _ := filepath.Abs(*dbPath)
	log.Printf("数据库文件: %s", abs)
	initDB(*dbPath)

	tpl = template.Must(template.ParseFS(tplFS, "templates/*.html"))

	mux := http.NewServeMux()
	realAdminPath := normalizeAdminPath(*adminPath)
	registerRoutes(mux, realAdminPath)

	log.Printf("发布页启动: http://%s  (后台路径: %s)", *addr, realAdminPath)
	srv := &http.Server{Addr: *addr, Handler: securityHeaders(mux)}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func staticHandler() http.Handler {
	sub, _ := fs.Sub(staticFS, "static")
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub)))
}

// securityHeaders 加基础安全响应头
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "no-referrer-when-downgrade")
		next.ServeHTTP(w, r)
	})
}
