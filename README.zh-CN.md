# xray-reality-solo-vpn

[English](./README.md) | 简体中文 | [日本語](./README.ja.md)

`xray-reality-solo-vpn` 是一个面向个人自建场景的单机安全接入与管理面板，`Solo VPN` 作为面板内的产品名使用。它服务的不是“多节点平台”场景，而是更真实的使用方式：一台 VPS、一个管理员、少量设备、一个稳定可控的自建接入环境。

## 产品边界

- 单机部署
- 首次初始化管理员
- 登录与会话管理
- 创建 / 停用 / 删除客户端
- 生成 `vless://` 分享链接
- 导出 Mihomo / Clash Meta 订阅
- 展示基础流量和最后活跃时间
- 重写 Xray 运行时配置并触发同步
- 在面板内修改管理员密码

它不是多节点控制台，不是面向公众的共享接入平台，也不是多租户分发 SaaS。

## 技术栈

- `Go` 管理后端
- `React + Vite + Tailwind + shadcn/ui 风格组件` 前端
- 内嵌 `SQLite`
- `Xray-core + VLESS + REALITY`
- `systemd + Caddy + Nginx stream` 负责宿主机部署和 `443/tcp` 分流

## 目录结构

- `cmd/manager/`
  Go 后端入口
- `internal/`
  配置、鉴权、HTTP API、SQLite、订阅与运行时同步
- `web/`
  前端源码和构建产物
- `scripts/install.sh`
  交互式宿主机安装脚本
- `scripts/check.sh`
  残留与端口检测
- `scripts/cleanup.sh`
  旧部署清理
- `deploy/`
  systemd、Caddy、Nginx 配置模板
- `scripts/bootstrap-reality-env.sh`
  生成 `XRAY_PRIVATE_KEY`、`XRAY_PUBLIC_KEY`、`SESSION_SECRET`
- `generated/server.json`
  生成后的 Xray 运行时配置
- `data/manager.db`
  SQLite 数据库

## 宿主机安装

在目标 Ubuntu VPS 上执行：

```bash
./scripts/check.sh
./scripts/install.sh
```

`install.sh` 会交互式询问必要信息，写入 `/etc/xray-reality-solo-vpn/app.env`，安装宿主机服务，并输出一次性 setup 链接。

## 部署产物要求

上传到服务器的项目目录必须已经包含这两个预构建产物：

- `web/dist/`
- `build/manager-linux-amd64`

推荐部署流程：

1. 在本地先构建前端和后端产物
2. 上传包含这两个产物的完整项目目录
3. 在服务器上执行：

```bash
./scripts/check.sh
./scripts/install.sh
```

不要依赖目标服务器在部署时补构建缺失的前端或后端产物。

## 宿主机服务

- `xray-reality-solo-vpn.service`
  Go API + 静态前端托管，监听 `127.0.0.1:3000`
- `xray.service`
  Reality 服务端，监听 `127.0.0.1:2443`
- `caddy.service`
  面板 HTTPS
- `nginx.service`
  `443/tcp` 的 SNI 分流

## 域名与地址

- `PANEL_DOMAIN`
  面板域名，例如 `panel.example.com`
- `LINE_DOMAIN`
  逻辑线路域名，用于 UI 展示和订阅内容
- `LINE_SERVER_ADDRESS`
  客户端实际拨号地址。若存在 fake-ip DNS、TUN 回环或本地代理递归，建议直接填服务器公网 IP

## Setup 流程

- 默认不公开初始化页
- 安装脚本会输出一次性链接：
  `https://<panel-domain>/_/setup/<token>`
- 只有访问该链接的浏览器才能授权首个管理员创建
- 初始化成功后该链接永久失效，后续统一使用 `/login`

## 本地开发

后端：

```bash
go test ./...
go run ./cmd/manager
```

前端：

```bash
npm --prefix web install
npm --prefix web run dev
npm --prefix web run build
```

现在不再支持 Docker / Compose 部署，统一使用 `scripts/install.sh` 配合 `deploy/` 下的宿主机模板进行安装。

## 部署说明

- 不要把 `.env`、`generated/`、`data/` 提交到 Git
- `panel.example.com` 和 `line.example.com` 通常应解析到同一台 VPS
- 如果服务端健康但线路仍不稳，优先排查 DNS、fake-ip、TUN 回环和 VPS 线路质量
- 面向中国大陆使用时，体验通常更受机房和线路质量影响，而不是面板代码

## 许可证

本项目采用双许可证发布，你可以任选其一使用：

- MIT（`LICENSE-MIT`）
- Apache License 2.0（`LICENSE-APACHE`）
