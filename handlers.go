package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func registerRoutes(mux *http.ServeMux, adminPath string) {
	// 静态资源
	mux.Handle("/static/", staticHandler())

	// 前台
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/api/links", handleAPILinks)
	mux.HandleFunc("/go/", handleRedirect)
	mux.HandleFunc("/blocked", handleBlocked)

	// 后台(路径可配置)
	adminPath = normalizeAdminPath(adminPath)
	mux.HandleFunc(adminPath, redirectTo(adminPath+"/"))
	mux.HandleFunc(adminPath+"/", makeAdminRouter(adminPath))
}

// normalizeAdminPath 规范化后台路径,非法时回退到 /manage
// 例如 "/" / "" / "manage" 等都会被纠正,避免 http: invalid pattern
func normalizeAdminPath(p string) string {
	p = strings.TrimSpace(p)
	p = "/" + strings.Trim(p, "/") // 去掉首尾斜杠再补一个,得到形如 /xxx
	if p == "/" {
		return "/manage"
	}
	return p
}

func redirectTo(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	}
}

// ---------- 前台 ----------

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s := allSettings()

	// 1) 内置浏览器拦截(后端兜底,前端还会再判一次)
	if isInAppBrowser(r.UserAgent()) {
		renderBlocked(w, s)
		return
	}

	// 2) 入口列表
	render(w, "index.html", map[string]any{
		"Title":         s["site_title"],
		"Subtitle":      s["site_subtitle"],
		"ThemeColor":    s["theme_color"],
		"Email":         s["contact_email"],
		"ICP":           s["icp"],
		"Tips":          s["tips"],
		"Copyright":     s["copyright"],
		"ChatwootURL":   s["chatwoot_base_url"],
		"ChatwootToken": s["chatwoot_token"],
	})
}

// handleAPILinks 前台拿线路列表
func handleAPILinks(w http.ResponseWriter, r *http.Request) {
	links := listLinks(true)
	type item struct {
		ID     int64  `json:"id"`
		Title  string `json:"title"`
		Remark string `json:"remark"`
		Host   string `json:"host"`
	}
	out := make([]item, 0, len(links))
	for _, l := range links {
		out = append(out, item{ID: l.ID, Title: l.Title, Remark: l.Remark, Host: hostOf(l.URL)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "links": out})
}

// handleRedirect /go/{id} -> 302 跳转到真实地址,并统计点击
func handleRedirect(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/go/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	l, ok := getLink(id)
	if !ok || l.Enabled != 1 {
		http.NotFound(w, r)
		return
	}
	incClick(id)
	http.Redirect(w, r, l.URL, http.StatusFound)
}

func handleBlocked(w http.ResponseWriter, r *http.Request) {
	renderBlocked(w, allSettings())
}

func renderBlocked(w http.ResponseWriter, s map[string]string) {
	render(w, "blocked.html", map[string]any{
		"Title":      s["site_title"],
		"ThemeColor": s["theme_color"],
	})
}

// ---------- 工具 ----------

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	noCache(w)
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	noCache(w)
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// noCache 禁止浏览器/CDN 缓存动态页面与接口,确保网址实时最新
func noCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func hostOf(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	return u
}
