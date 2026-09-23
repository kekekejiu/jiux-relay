package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// makeAdminRouter 返回挂载在 adminPath+"/" 下的处理器
func makeAdminRouter(adminPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 子路径,去掉前缀
		sub := strings.TrimPrefix(r.URL.Path, adminPath+"/")
		ctx := &adminCtx{base: adminPath, w: w, r: r}

		switch {
		case sub == "" || sub == "login":
			if sub == "login" {
				ctx.login()
				return
			}
			ctx.requireAuth(ctx.dashboard)
		case sub == "logout":
			ctx.logout()
		case sub == "links/save":
			ctx.requireAuth(ctx.linkSave)
		case sub == "links/delete":
			ctx.requireAuth(ctx.linkDelete)
		case sub == "settings":
			ctx.requireAuth(ctx.settingsSave)
		case sub == "password":
			ctx.requireAuth(ctx.passwordChange)
		default:
			http.NotFound(w, r)
		}
	}
}

type adminCtx struct {
	base string
	w    http.ResponseWriter
	r    *http.Request
}

func (c *adminCtx) requireAuth(next func(user string)) {
	user, ok := adminUser(c.r)
	if !ok {
		http.Redirect(c.w, c.r, c.base+"/login", http.StatusFound)
		return
	}
	next(user)
}

func (c *adminCtx) login() {
	if c.r.Method == http.MethodGet {
		if _, ok := adminUser(c.r); ok {
			http.Redirect(c.w, c.r, c.base+"/", http.StatusFound)
			return
		}
		render(c.w, "admin_login.html", map[string]any{"Base": c.base, "Err": ""})
		return
	}
	// POST
	if !loginLimiter.allow("admin:"+clientIP(c.r), 8, 5*time.Minute) {
		render(c.w, "admin_login.html", map[string]any{"Base": c.base, "Err": "尝试过于频繁，请 5 分钟后再试"})
		return
	}
	u := strings.TrimSpace(c.r.FormValue("username"))
	p := c.r.FormValue("password")
	if checkAdmin(u, p) {
		setCookie(c.w, authCookie, u, 12*time.Hour)
		http.Redirect(c.w, c.r, c.base+"/", http.StatusFound)
		return
	}
	render(c.w, "admin_login.html", map[string]any{"Base": c.base, "Err": "用户名或密码错误"})
}

func (c *adminCtx) logout() {
	clearCookie(c.w, authCookie)
	http.Redirect(c.w, c.r, c.base+"/login", http.StatusFound)
}

func (c *adminCtx) dashboard(user string) {
	render(c.w, "admin_dashboard.html", map[string]any{
		"Base":     c.base,
		"User":     user,
		"Links":    listLinks(false),
		"Settings": allSettings(),
		"Msg":      c.r.URL.Query().Get("msg"),
	})
}

func (c *adminCtx) linkSave(user string) {
	if c.r.Method != http.MethodPost {
		http.Redirect(c.w, c.r, c.base+"/", http.StatusFound)
		return
	}
	id, _ := strconv.ParseInt(c.r.FormValue("id"), 10, 64)
	sort, _ := strconv.Atoi(c.r.FormValue("sort"))
	enabled := 0
	if c.r.FormValue("enabled") == "1" {
		enabled = 1
	}
	l := Link{
		ID:      id,
		Title:   strings.TrimSpace(c.r.FormValue("title")),
		URL:     strings.TrimSpace(c.r.FormValue("url")),
		Remark:  strings.TrimSpace(c.r.FormValue("remark")),
		Sort:    sort,
		Enabled: enabled,
	}
	if l.Title == "" || l.URL == "" {
		http.Redirect(c.w, c.r, c.base+"/?msg=标题和网址不能为空", http.StatusFound)
		return
	}
	if id > 0 {
		updateLink(l)
	} else {
		createLink(l)
	}
	http.Redirect(c.w, c.r, c.base+"/?msg=已保存", http.StatusFound)
}

func (c *adminCtx) linkDelete(user string) {
	id, _ := strconv.ParseInt(c.r.FormValue("id"), 10, 64)
	if id > 0 {
		deleteLink(id)
	}
	http.Redirect(c.w, c.r, c.base+"/?msg=已删除", http.StatusFound)
}

func (c *adminCtx) settingsSave(user string) {
	if c.r.Method != http.MethodPost {
		http.Redirect(c.w, c.r, c.base+"/", http.StatusFound)
		return
	}
	c.r.ParseForm()
	keys := []string{"site_title", "site_subtitle", "access_password", "contact_email", "icp", "tips", "theme_color", "copyright",
		"chatwoot_base_url", "chatwoot_token"}
	for _, k := range keys {
		if v := c.r.Form.Get(k); c.r.Form.Has(k) {
			setSetting(k, strings.TrimSpace(v))
		}
	}
	http.Redirect(c.w, c.r, c.base+"/?msg=设置已更新", http.StatusFound)
}

func (c *adminCtx) passwordChange(user string) {
	if c.r.Method != http.MethodPost {
		http.Redirect(c.w, c.r, c.base+"/", http.StatusFound)
		return
	}
	np := strings.TrimSpace(c.r.FormValue("new_password"))
	if len(np) < 6 {
		http.Redirect(c.w, c.r, c.base+"/?msg=新密码至少6位", http.StatusFound)
		return
	}
	changeAdminPassword(user, np)
	clearCookie(c.w, authCookie)
	http.Redirect(c.w, c.r, c.base+"/login", http.StatusFound)
}
