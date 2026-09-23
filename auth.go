package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 会话密钥:进程启动时随机生成(重启后旧 cookie 失效,可接受)
var sessionSecret = genSecret()

func genSecret() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}

// sign 生成 value|exp|hmac 形式的签名串
func sign(value string, ttl time.Duration) string {
	exp := strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	payload := value + "|" + exp
	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
}

// verify 校验签名串,返回 value 与是否有效
func verify(token string) (string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return "", false
	}
	value, exp, sig := parts[0], parts[1], parts[2]
	payload := value + "|" + exp
	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(payload))
	expect := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expect)) {
		return "", false
	}
	expUnix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || time.Now().Unix() > expUnix {
		return "", false
	}
	return value, true
}

func setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    sign(value, ttl),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", HttpOnly: true,
		Expires: time.Unix(0, 0), MaxAge: -1,
	})
}

func cookieValid(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return verify(c.Value)
}

const (
	gateCookie = "relay_gate"  // 通过密码门
	authCookie = "relay_admin" // 管理员登录
)

func gatePassed(r *http.Request) bool {
	_, ok := cookieValid(r, gateCookie)
	return ok
}

func adminUser(r *http.Request) (string, bool) {
	return cookieValid(r, authCookie)
}

// --- 简单登录失败限流(防爆破) ---
type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

var loginLimiter = &rateLimiter{attempts: map[string][]time.Time{}}

func (rl *rateLimiter) allow(key string, max int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	var kept []time.Time
	for _, t := range rl.attempts[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	rl.attempts[key] = kept
	return len(kept) <= max
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return xr
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}

// isInAppBrowser 检测微信/QQ等内置浏览器
func isInAppBrowser(ua string) bool {
	ua = strings.ToLower(ua)
	keys := []string{"micromessenger", "qq/", "qqbrowser", "weibo", "dingtalk", "alipayclient", "ucbrowser"}
	for _, k := range keys {
		if strings.Contains(ua, k) {
			return true
		}
	}
	return false
}

func debugf(format string, a ...any) {
	fmt.Printf("[debug] "+format+"\n", a...)
}
