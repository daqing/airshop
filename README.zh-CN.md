# AirShop

一个基于 Go、构建在 [Airway](https://github.com/daqing/airway) 框架上的
完整开源电商平台。
英文版见 [README.md](README.md),两份内容一一对应。

## 项目简介

AirShop 用 Go 实现了一个完整的电商系统——服务端渲染、数据库支撑,并可打包为
原生桌面应用。项目正在积极开发中。

当前范围聚焦于经典的电商核心功能;AI 功能将在第三阶段提供(待规划)。

## 路线图

- **阶段一:核心电商**(*已完成*):完整核心电商功能已实现——商品目录、购物车、
  结算、订单、支付、优惠券、手机号登录的账户中心、地址簿、物流追踪与管理后台。
- **阶段二:扩展**:开发转向 AirShop 特有的功能。
- **阶段三:AI 功能**(*待规划*):本阶段提供 AI 相关功能。

## 功能

商店前台:

- 首页展示最新商品
- 商品目录:分类筛选与详情
- 购物车(重复加购自动合并、库存上限)
- 结算:地址快照与金额服务端重算
- 订单:状态机与物流时间线
- 支付:网关注册表(内置本地假网关;微信支付与支付宝规划中)
- 优惠券:领券中心领取、结算使用
- 账户中心:手机号(短信验证码)登录
- 地址簿:省市区三级联动
- 物流追踪时间线
- 亦可打包为原生桌面应用(见下文)

管理后台:

- 独立管理员登录(用户名/密码)
- 仪表盘:销售额与待办统计
- 分类、商品(含图片与变体)、优惠券管理
- 订单管理与退款操作
- 发货与手动轨迹录入
- 高风险操作审计日志

## 技术栈

- **Go** + [Airway](https://github.com/daqing/airway) 框架——服务端渲染的
  Web 应用,自带数据库迁移、脚手架代码生成和项目 REPL 的 CLI。
- **数据库**:PostgreSQL、MySQL 或 SQLite;可选 Redis。
- **文件存储**:本地磁盘、Amazon S3、Cloudflare R2 或腾讯云 COS。
- **桌面端**:同一套应用可通过 Wails v3 打包为 macOS / Windows / Linux
  原生应用。

## 快速开始

脚手架已自动生成 `.env`——打开它,设置 `AIRWAY_ENV`(如 `local`)、`DSN` 和
`LISTEN` 监听地址(`host:port`,如 `:1900`):

```bash
airway db:create
airway db:migrate
go run .               # 启动 HTTP 服务(等价:airway server)
```

## 常用命令

```bash
airway generate api admin          # 生成一个 API 命名空间
airway generate model post         # 生成一个模型
airway generate migration create_posts
airway db:migrate
go run . repl                      # 带本项目模型的 REPL
```

## 桌面应用(macOS / Windows / Linux)

本项目可以打包为原生桌面应用:同一套 Web 技术栈运行在桌面进程内的本地端口
上,原生 WebView 窗口直接加载它——服务端渲染、cookie 会话、重定向、
WebSocket 的行为与线上完全一致,应用代码零改造。

### 1. 生成桌面目标

```bash
airway desktop:init
```

该命令会创建 `desktop/` 目录,即完整的 Wails v3 工程:窗口引导代码、内嵌的
`db/migrate` SQL(首次启动自动迁移)、插件镜像,以及三个平台的构建资产;
同时会把 `github.com/wailsapp/wails/v3` 固定到 `go.mod`。重复执行是安全的:
它只重新同步迁移与插件,不会改动 `desktop/main.go`(需要重新生成时加
`--force`)。

### 2. 安装桌面工具链(一次性)

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
go install github.com/go-task/task/v3/cmd/task@latest
```

各平台构建要求:

| 平台 | 要求 |
|---|---|
| macOS 12+ | Xcode 命令行工具(`xcode-select --install`) |
| Windows 10/11 | 构建无需额外依赖;安装器自带 WebView2 引导程序 |
| Linux | GTK4 + WebKitGTK 6.0 开发包(Ubuntu 24.04+ / Debian 13+),例如 `sudo apt install libgtk-4-dev libwebkitgtk-6.0-dev` |

### 3. 开发运行

```bash
cd desktop
wails3 task dev
```

### 4. 构建安装包

所有产物输出到 `desktop/bin/`。

#### macOS(.app)——需在 macOS 上构建

```bash
cd desktop
wails3 task package                    # 当前架构的 .app
wails3 task darwin:package:universal   # universal .app(Apple Silicon + Intel)
```

`.app` 默认 ad-hoc 签名,本机使用足够。要分发给他人,需用 Developer ID
证书签名并公证(先用 `wails3 setup` 完成一次性配置):

```bash
wails3 task darwin:sign:notarize
```

#### Windows(.exe + NSIS 安装器)——需在 Windows 或 CI 上构建

```bash
cd desktop
wails3 task package
```

产出 `bin/<app>.exe` 以及一个 NSIS 安装器;目标机器缺 WebView2 运行时时,
安装器会自动安装。构建安装器需要安装
[NSIS](https://nsis.sourceforge.io)。正式发布请用 Authenticode 证书签名
安装器:

```bash
wails3 task windows:sign:installer
```

#### Linux(deb / rpm / AppImage)——需在 Linux 上构建

```bash
cd desktop
wails3 task package
```

构建二进制,并通过 nfpm 产出 deb、rpm 包和 AppImage(AppImage 步骤首次运行
会下载 `linuxdeploy` 工具)。deb/rpm 声明 GTK4 + WebKitGTK 6.0 依赖;老发行
版用旧栈构建(`EXTRA_TAGS=gtk3`),并相应调整
`desktop/build/linux/nfpm/nfpm.yaml`。

### 5. 交叉编译与 CI

- Windows 可执行文件可从 macOS/Linux 免 CGO 交叉编译:
  `wails3 task build GOOS=windows`。
- macOS 和 Linux 构建依赖 CGO,正式发布应在目标 OS 上进行——推荐 GitHub
  Actions 三平台矩阵(每个平台一个 job),或使用 Wails 官方 Docker 交叉
  镜像(`wails3 task setup:docker`,约 800MB)。
- 交叉编译出的产物不带签名,分发前需在目标 OS 上完成签名。

### 运行时数据与迁移

桌面构建每次启动都会自动执行迁移。数据存放在用户配置目录:macOS 为
`~/Library/Application Support/<name>`,Windows 为 `%APPDATA%\<name>`,
Linux 为 `~/.local/share/<name>`。项目新增迁移或插件后,重跑
`airway desktop:init` 刷新内嵌副本。

框架的[桌面指南](https://github.com/daqing/airway/blob/main/docs/zh-CN/desktop.md)
包含完整的原理说明与注意事项。

## 致谢

- [Airway](https://github.com/daqing/airway)——本项目所基于的 Go Web 框架。

## 许可证

AirShop 采用 [MIT 许可证](LICENSE)发布。
