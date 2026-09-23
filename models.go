package main

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

// Link 前台/后台共用的线路结构
type Link struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Remark  string `json:"remark"`
	Sort    int    `json:"sort"`
	Enabled int    `json:"enabled"`
	Clicks  int    `json:"clicks"`
}

func seedData() {
	// 默认设置
	defaults := map[string]string{
		"site_title":      "极速发布页",
		"site_subtitle":   "最新永久发布页",
		"access_password": "666",
		"contact_email":   "your-email@example.com",
		"icp":             "",
		"copyright":       "Copyright © 2019-2026 本站版权所有",
		"tips":            "推荐使用谷歌(Chrome)浏览器访问本站，速度更快；iPhone 建议使用 Safari 访问。\n如果记不住本站域名，请收藏该页地址，并分享给好朋友。\n1、电脑用户：按键盘 Ctrl+D 收藏本页。\n2、苹果手机：在浏览器点击分享，添加到个人收藏或主屏幕。\n3、安卓手机：点击菜单，添加到书签或主屏幕。",
		"theme_color":     "#10b981",
		// 在线客服(Chatwoot)。留空则前台不加载客服脚本。
		"chatwoot_base_url": "",
		"chatwoot_token":    "",
	}
	for k, v := range defaults {
		db.Exec(`INSERT OR IGNORE INTO settings(k, v) VALUES(?, ?)`, k, v)
	}

	// 默认管理员 admin / admin888 (首次登录后请立即改密)
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n)
	if n == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin888"), bcrypt.DefaultCost)
		_, err := db.Exec(`INSERT INTO admins(username, password_hash) VALUES(?, ?)`, "admin", string(hash))
		if err != nil {
			log.Printf("seed admin: %v", err)
		} else {
			log.Println("已创建默认管理员: admin / admin888 (请尽快修改)")
		}
	}

	// 示例线路
	var lc int
	db.QueryRow(`SELECT COUNT(*) FROM links`).Scan(&lc)
	if lc == 0 {
		samples := []Link{
			{Title: "移动 联通 国内入口一", URL: "https://www.example.com", Remark: "此链接随时更新", Sort: 1},
			{Title: "北上广 官网入口二", URL: "https://www.example.com", Remark: "此链接随时更新", Sort: 2},
			{Title: "电信 官网入口三", URL: "https://www.example.com", Remark: "此链接随时更新", Sort: 3},
		}
		for _, s := range samples {
			db.Exec(`INSERT INTO links(title, url, remark, sort, enabled) VALUES(?,?,?,?,1)`,
				s.Title, s.URL, s.Remark, s.Sort)
		}
	}
}

func getSetting(k string) string {
	var v string
	db.QueryRow(`SELECT v FROM settings WHERE k=?`, k).Scan(&v)
	return v
}

func setSetting(k, v string) error {
	_, err := db.Exec(`INSERT INTO settings(k,v) VALUES(?,?)
		ON CONFLICT(k) DO UPDATE SET v=excluded.v`, k, v)
	return err
}

func allSettings() map[string]string {
	m := map[string]string{}
	rows, err := db.Query(`SELECT k, v FROM settings`)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		m[k] = v
	}
	return m
}

// listLinks onlyEnabled=true 时只返回已启用的(前台用)
func listLinks(onlyEnabled bool) []Link {
	q := `SELECT id, title, url, remark, sort, enabled, clicks FROM links`
	if onlyEnabled {
		q += ` WHERE enabled=1`
	}
	q += ` ORDER BY sort ASC, id ASC`
	rows, err := db.Query(q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Link
	for rows.Next() {
		var l Link
		rows.Scan(&l.ID, &l.Title, &l.URL, &l.Remark, &l.Sort, &l.Enabled, &l.Clicks)
		out = append(out, l)
	}
	return out
}

func getLink(id int64) (Link, bool) {
	var l Link
	err := db.QueryRow(`SELECT id, title, url, remark, sort, enabled, clicks FROM links WHERE id=?`, id).
		Scan(&l.ID, &l.Title, &l.URL, &l.Remark, &l.Sort, &l.Enabled, &l.Clicks)
	return l, err == nil
}

func createLink(l Link) error {
	_, err := db.Exec(`INSERT INTO links(title, url, remark, sort, enabled) VALUES(?,?,?,?,?)`,
		l.Title, l.URL, l.Remark, l.Sort, l.Enabled)
	return err
}

func updateLink(l Link) error {
	_, err := db.Exec(`UPDATE links SET title=?, url=?, remark=?, sort=?, enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		l.Title, l.URL, l.Remark, l.Sort, l.Enabled, l.ID)
	return err
}

func deleteLink(id int64) error {
	_, err := db.Exec(`DELETE FROM links WHERE id=?`, id)
	return err
}

func incClick(id int64) {
	db.Exec(`UPDATE links SET clicks = clicks + 1 WHERE id=?`, id)
}

func checkAdmin(username, password string) bool {
	var hash string
	err := db.QueryRow(`SELECT password_hash FROM admins WHERE username=?`, username).Scan(&hash)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func changeAdminPassword(username, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE admins SET password_hash=? WHERE username=?`, string(hash), username)
	return err
}
