# 网址中转 / 域名发布页

Go + SQLite 单文件应用。功能：微信/QQ 内置浏览器拦截、多线路入口 + 前端延迟检测、302 中转点击统计、Admin 后台热更新网址。

## 一、目录结构

```
relay/
├── relay              # 编译好的二进制(直接运行)
├── main.go            # 入口、路由注册、embed 静态资源
├── handlers.go        # 前台:UA拦截/入口API/302中转
├── admin.go           # 后台:登录/线路CRUD/设置/改密码
├── auth.go            # 签名Cookie会话、限流、UA检测
├── db.go              # SQLite 连接与建表
├── models.go          # 数据访问层 + 初始种子数据
├── go.mod / go.sum
├── templates/         # HTML 模板(已 embed 进二进制)
└── static/            # CSS/JS(已 embed 进二进制)
```

> 编译后的 `relay` 已把 templates 和 static 打包进去，**部署时只需要这一个二进制文件**，不依赖源码目录。

## 二、运行参数

| 参数 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| `-addr` | `RELAY_ADDR` | `127.0.0.1:8080` | 监听地址 |
| `-db` | `RELAY_DB` | `app.db` | SQLite 文件路径 |
| `-admin` | `RELAY_ADMIN_PATH` | `/manage` | 后台路径(建议改隐蔽路径) |

示例：

```bash
./relay -addr 127.0.0.1:8080 -db /www/wwwroot/relay/app.db -admin /x8s9k2
```

## 三、默认账号(首次启动自动创建)

- 后台地址：`你的域名 + 后台路径`（如 `https://xxx.com/manage`）
- 管理员：`admin` / `admin888`

**首次登录后务必：1) 改后台密码 2) 改后台路径。**

## 四、宝塔部署步骤

### 1. 上传二进制
把 `/www/wwwroot/relay/relay` 留在服务器即可（已编译好）。

### 2. 用 Supervisor 守护进程
宝塔软件商店安装「**Supervisor 管理器**」，添加守护进程：

- 名称：`relay`
- 启动用户：`root`（或 www）
- 运行目录：`/www/wwwroot/relay`
- 启动命令：`/www/wwwroot/relay/relay -addr 127.0.0.1:8080 -db /www/wwwroot/relay/app.db -admin /你的隐蔽路径`

保存后进程会开机自启、崩溃自动拉起，日志在 Supervisor 面板里看。

### 3. Nginx 反向代理
宝塔「网站」新建站点，绑定你的域名，申请 SSL，然后在「反向代理」里：

- 目标 URL：`http://127.0.0.1:8080`
- 发送域名：`$host`

或手动加 Nginx 配置：

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

# 禁止外部直接下载数据库文件
location ~ \.db {
    deny all;
}
```

## 五、日常使用

- **改网址(热更新)**：登录后台 → 线路管理 → 编辑/新增/删除 → 保存。用户刷新页面立即看到新网址，无需重启。
- **改标题 / 提示语 / 主题色**：后台「全局设置」。
- **在线客服（可选）**：后台「全局设置」填写 Chatwoot 地址与 websiteToken，两项都填才会启用；留空则前台不加载任何客服脚本。
- **备份**：备份 `app.db` 一个文件即可（宝塔文件管理器可下载）。

## 六、重新编译(改了源码后)

```bash
cd /www/wwwroot/relay
export PATH=$PATH:/usr/local/go/bin
export GOTOOLCHAIN=local
CGO_ENABLED=0 go build -o relay .
# 然后在 Supervisor 里重启 relay 进程
```

## 七、安全要点

- 管理员密码后端校验，不出现在前台源码里。
- 管理员密码用 bcrypt 哈希存储。
- 后台登录有同 IP 限流，防爆破。
- 密码页对未授权请求只返回密码页，不暴露真实入口，挡爬虫。
- 后台路径可自定义为隐蔽路径，降低被扫描概率。
- `app.db` 含真实网址与管理员密码哈希，已在 `.gitignore` 中排除，**不要提交到 Git 仓库**。
- 首次部署请立即修改默认管理员密码与后台路径。
