# AirShop 开发任务清单

基于 README 的功能范围拆解的开发任务,按依赖顺序排列,供闲余时间分步推进。
英文版见 [TASKS.md](TASKS.md),两份内容一一对应。

## 使用说明

- 每个任务都有编号(`T<里程碑>.<序号>`,如 T0.1);在 commit、记录或沟通中引用任务时用编号。
- 完成一个任务就把 `[ ]` 改成 `[x]`;顺序可以调整,但注意每个里程碑开头的**前置依赖**。
- 粒度按"一个闲余时间段(约 1–3 小时)完成一个任务"设计;大任务拆成了子任务。
- 每完成一个任务,跑一遍 `go test ./...`,里程碑完成时顺手记一行日期。
- 带 ❓ 的是**待决策项**:实现前先定方案,定了以后把结论写在任务后面,避免反复。

## 工作流速查

```bash
airway generate model post         # 生成模型
airway generate migration create_posts
airway db:migrate                  # 执行迁移
airway generate api admin          # 生成 API 命名空间
just generate                      # 由 .templ 重新生成 *_templ.go
go run .                           # 启动服务
go run . repl                      # 带项目模型的 REPL
```

约定:`app/api/<name>_api/` 放路由+action,`app/views/` 放 templ 视图,`app/models/` 放模型(记得注册进 REPL),`app/services/` 放业务逻辑,`db/migrate/` 放迁移 SQL。

---

## M0 基础设施

前置依赖:无(当前代码为脚手架基线)。

- [x] T0.1 本地环境跑通:`airway db:create && airway db:migrate && go run .`,确认首页可访问 —— 2026-09-28 完成:本机 Postgres(127.0.0.1:5432),创建数据库 `airshop`,迁移通过(生成 `db/schema.json`),首页 HTTP 200(`/health` 200)。DSN 通过 `AIRWAY_DSN` 环境变量传入(进程环境优先于 `.env`);要长期生效请在 `.env` 里设置 `DSN`
- [x] T0.2 ❓ 定公共字段约定:主键类型、`created_at`/`updated_at`、软删除与否 —— 2026-09-28 已定,详见[CONVENTIONS.zh-CN.md](CONVENTIONS.zh-CN.md):自增整数 `id`(int64)、脚手架标准时间戳、显式维护 `updated_at`、硬删除 + 状态字段替代、不做数据库级外键
- [x] T0.3 前后台布局分离:`app/views/layouts/` 增加 storefront 与 admin 两套布局(顶栏、导航、页脚骨架)—— 2026-09-28 完成:`storefront.templ`(品牌顶栏、导航、页脚)和 `admin.templ`(后台顶栏、导航)均套在 `Base` 之上;首页已接入 Storefront;覆盖测试见 `app/views/layouts/layouts_test.go`
- [x] T0.4 路由分区:前台路由与 `/admin` 后台路由分组,404/403 兜底页面 —— 2026-09-28 完成:`AdminRoutes` 组挂在 `/admin`(仪表盘占位页,套 admin 布局);`NoRoute` 渲染 storefront 风格 404 页;导出 `ForbiddenHandler` 渲染 403 页,供后续鉴权中间件使用;覆盖测试见 `config/routes_test.go`
- [x] T0.5 ❓ 金额与库存的存储口径定版(建议金额用最小货币单位的整数,如"分"),后续所有表统一 —— 2026-09-28 已定,详见[CONVENTIONS.zh-CN.md](CONVENTIONS.zh-CN.md):金额用 BIGINT 最小货币单位(`_cents` 后缀,Go 侧 int64),单一门店货币,库存为非负 INTEGER,百分比为整数 0–100

## M1 商品目录

前置依赖:M0。

### 数据层

- [x] T1.1 迁移:`categories` 表(名称、slug、父分类、排序、启用)—— 2026-09-28 完成:首个按 T0.2/T0.5 约定写的迁移(`BIGSERIAL` 主键、`slug` UNIQUE、`parent_id` 可空 BIGINT 不带外键、`sort_order` INTEGER、`enabled` BOOLEAN、标准时间戳);用 `db:migrate` + `db:rollback` 在本机 Postgres 上完成 up/down 往返验证
- [x] T1.2 迁移:`products` 表(名称、slug、描述、价格、库存、状态:上架/下架、主图)—— 2026-09-28 完成:遵循 T0.2/T0.5(`price_cents` BIGINT、`stock` INTEGER、`active` BOOLEAN 对齐 categories.enabled、`main_image` VARCHAR 存储键、`description` TEXT NOT NULL DEFAULT '');另加可空带索引的 `category_id`(供 T1.9 分类筛选);up/down 往返已在本机 Postgres 验证
- [x] T1.3 ❓ 是否需要 SKU 变体(`product_variants`:规格、独立价格库存)?先定,影响表结构 —— 2026-09-28 由 David Zhang 拍板:**现在就支持变体**;`product_variants` 迁移已建并验证(独立 `price_cents`/`stock`、组合名 `name`、启用位);可售单元规则(有启用变体走变体,否则走商品)收在 Product 服务层;结构化选项轴后置;详见 [CONVENTIONS.zh-CN.md](CONVENTIONS.zh-CN.md) "商品变体"
- [x] T1.4 模型与 REPL 注册:`Category`、`Product`(及变体)—— 2026-09-28 完成:三个模型按脚手架样式落在 `app/models/`(db/json tag、`TableName()`、REPL 注册);可空列 `parent_id`/`category_id` 用 `*int64`;可售单元判定按 T1.3 规则落在 `app/services/product_service.go`;main.go 以 blank import 把模型注册挂进二进制;已用 `go run . repl` 往返查询三个模型的样例行做端到端验证

### 后台管理

- [x] T1.5 admin 分类管理:列表、新建、编辑、删除 —— 2026-09-28 完成:`services/category_service.go`(校验:名称必填、slug 格式/唯一、父分类存在、禁止自引用、有子分类时禁止删除;按 T0.2 约定维护 `updated_at`);`/admin/categories` 下的服务端渲染管理页,套 admin 布局;已起真实服务端到端验证(建父子分类、编辑表单回填、更新、重复 slug 422、自引用 422、删除保护重定向、删除)
- [x] T1.6 admin 商品管理:列表(分页、按名称/状态筛选)、新建、编辑、上下架 —— 2026-09-28 完成:`services/product_admin_service.go`(名称/slug 校验、slug 可由名称自动生成、分类存在性、价格库存非负;按 T0.2 维护 `updated_at`)+ `services/money.go` 共享金额格式化(T0.5 首次落地);`/admin/products` 下的 admin 页面,套 admin 布局;端到端已验证(自动 slug 创建、编辑回填、改价、ILIKE 名称搜索、状态筛选、组合筛选、下架/上架、重复 slug/非法价格/缺名称 422、21 条数据分页)
- [ ] T1.6a 变体管理:在商品表单内编辑变体(增删行:名称、价格、库存、启用、排序)—— T1.6 期间补记:T1.3 定了变体方案,但原清单漏了这项
- [x] T1.7 商品图片上传(对接已有 storage,支持多图,第一张为主图)—— 2026-09-28 完成:`product_images` 表 + 模型;商品编辑页多文件上传(单文件上限 5MB,仅 jpg/png/webp/gif),经 airway 存储层落盘在 `products/` 下;第一张即主图,`products.main_image` 保持同步(商品保存不再覆盖它);设主图重排,删除主图自动递补;端到端已验证(传两张、落盘 + URL 可访问、设主图、删主图递补、删空清主图、非图片拒绝)

### 前台

- [x] T1.8 首页改造:商品展示区(推荐位或最新商品)—— 2026-09-29 完成:storefront 首页替换脚手架欢迎页;最新 8 个上架商品以卡片展示(主图缩略或占位图、格式化价格、链接到 `/products/<slug>`);`services.LatestProducts` 由环境门控的集成测试验证(`AIRWAY_PG_TEST_DSN`,在 scratch 库上跑真实迁移);首页端到端已验证(空态、仅上架、最新在前、价格格式化)
- [x] T1.9 商品列表页:分类筛选、分页 —— 2026-09-29 完成:`/products` 前台页面(每页 12、仅上架),分类 chip(只列启用分类、当前高亮、未知 slug 回退全部)+ Prev/Next 分页;商品卡片抽成共享组件 `views/catalog.Grid`,首页同步复用;`services.StorefrontListProducts` 负责筛选 + 计数;端到端已验证(全部视图、分类筛选、第 2 页、下架不出现、停用分类不显示、未知分类回退)
- [x] T1.10 商品详情页:图片、价格、库存、描述、加购按钮(按钮先占位,M4 接通)—— 2026-09-29 完成:`/products/<slug>`(仅上架商品;未知或下架 slug 渲染 storefront 404 页);图廊(主图 + 缩略图,无图用 CSS 占位),按可售单元规则显示价格块或变体单选组,库存标签,描述,禁用态加购按钮;视图测试 + 端到端覆盖(简单商品、psql 种子的变体商品、两类 404)

## M2 用户与认证(手机号注册登录)

前置依赖:M0。

- [x] T2.1 迁移:`users` 表(手机号唯一、密码哈希、昵称、状态)—— 2026-09-29 完成:`phone` VARCHAR(20) UNIQUE(容纳 E.164),`password_hash` VARCHAR(255) NOT NULL DEFAULT ''(纯验证码登录时保持为空,哈希算法 T2.2 定),`display_name` VARCHAR(255) DEFAULT '',`status` VARCHAR(20) DEFAULT 'active'(active/disabled,留扩展余地);标准时间戳;up/down 往返已在本机 Postgres 验证
- [x] T2.2 ❓ 登录方式定版 —— 2026-09-29 由 David Zhang 拍板:**仅手机号 + 短信验证码登录,不做密码**。本地开发用假验证码(固定值/日志输出),短信服务商后置。`password_hash` 列保留(默认空)以备将来;同决策将 T2.1 迁移的列名 `phone` 改为 `phone_number`。
- [x] T2.3 注册页/登录页/登出 action —— 2026-09-29 完成:登录与注册合并为手机号+验证码单流程(`/signin` 两步表单,新用户自动创建为 "User <尾号4位>");`POST /signin/code`(按 T2.2 决策验证码打印到服务端日志,内存存储,5 分钟有效,单次使用)、`POST /signin`、`POST /signout`(删会话行 + 清 cookie);会话机制随本任务提前落地(`sessions` 表 + token cookie,T2.4 的中间件仍未做);禁用账号拒绝登录。服务层测试 + 完整 E2E 覆盖(非法手机号、错码、日志取码登录、自动注册、登出、二次登录复用用户)
- [x] T2.4 会话:cookie session 落地,登录态中间件(未登录访问受保护页面跳登录)—— 2026-09-29 完成:会话写入侧已随 T2.3 落地(`sessions` 表 + httpOnly token cookie);`middlewares.LoadUser` 每请求解析 cookie,`RequireUser` 将访客跳到 `/signin?next=<路径>` 且登录后同站安全回跳;`/account` 受保护页(资料 + 登出);storefront 导航按登录态切换 Sign in / 账号链接。layouts 测试(两种导航态)+ E2E 覆盖(访客重定向、next 往返、登录态导航、账户页、登出)
- [x] T2.5 表单校验与错误提示(手机号格式、重复注册)—— 2026-09-29 完成:大部分已随 T2.3 落地(手机号规范化/格式校验带用户可读错误、自动注册使"重复注册"在设计上不可能且有唯一约束兜底、登录页错误横幅经 templ 自动转义);本任务补充了空验证码的专用提示(`ErrCodeRequired`,在任何存储查询前返回);框架的 `lib/validation`(仅 required/email)评估后不适用,保留自定义手机号校验
- [x] T2.6 服务层:`AuthService`(注册/登录/登出),action 只做参数绑定 —— 2026-09-29 完成:认证逻辑全部收敛在 `services/auth_service.go`(手机号规范化、发码/验码含单次使用 + TTL + 过期清理、注册或登录、会话签发/解析/销毁且过期行自清理);action 只做参数绑定与重定向编排。收口审计中修掉一个真实时区 bug(本地时间写入 vs pgx UTC 标签读回,会话过期判断最多偏移 8 小时——时间写入已全部改 UTC,记入 CONVENTIONS);完整流程由环境门控集成测试(`AIRWAY_PG_TEST_DSN`)覆盖

## M3 地址簿

前置依赖:M2(地址挂在用户下)。

- [x] T3.1 迁移:`addresses` 表(用户 ID、收件人、手机号、省/市/区、详细地址、是否默认)—— 2026-09-29 完成:`user_id` BIGINT NOT NULL 带索引(按 T0.2 不带外键),省市区列为空默认 VARCHAR,使 T3.3 的数据来源决策后置也不需要改表,`is_default` BOOLEAN;"每用户仅一个默认"由服务层保证(T3.2),因部分唯一索引非三库可移植;up/down 往返已在本机 Postgres 验证
- [x] T3.2 用户中心地址 CRUD:列表、新增、编辑、删除、设默认 —— 2026-09-29 完成:`services/address_service.go`(收件人/手机号/街道校验,所有权全域强校验、跨用户访问一律 not-found,首地址强制默认,显式设默认清旧默认,删默认自动递补最新,`updated_at` 按 T2.6 用 UTC);`/account/addresses` 页面挂在 `RequireUser` 下(卡片 + DEFAULT 徽标 + 表单,省市区暂为自由文本待 T3.3);环境门控集成测试覆盖默认语义与跨用户隔离;E2E 已验证(建/列表/编辑/设默认/删除递补,校验 422)
- [x] T3.3 ❓ 省市区数据来源 —— 2026-09-29 由 David Zhang 拍板:**内置数据集**。`app/regions` 内嵌三级民政部区划数据(31 省 / 342 市 / 3056 区,137 KB,来自 modood/Administrative-divisions-of-China,MIT;港澳台暂未包含),并提供 `GET /api/v1/regions` 与 `GET /api/v1/regions/:code/children` 两个 JSON 端点。地址表单的三个自由文本输入升级为三级联动下拉(服务端按已存名称预选中,内联脚本在切换时拉取子级;直辖市与源数据一致地呈现"市辖区"层级)。数据集按名称解析,addresses 表结构无需变动。

## M4 购物车

前置依赖:M1、M2。

- [x] T4.1 ❓ 游客购物车策略 —— 2026-09-29 由 David Zhang 拍板:**仅登录用户可用**。不做游客购物车、不做合并逻辑;购物车归属用户账号(以 `user_id` 关联,T4.2)。游客点加购跳转 `/signin?next=<商品页>`,登录后回到商品页。连带效果:购物车页与结算都挂在 `RequireUser` 下,M5 结算直接读购物车。
- [x] T4.2 迁移:`carts` + `cart_items`(商品/变体、数量、加入时价格快照)—— 2026-09-29 完成:`carts` 以 `user_id` UNIQUE 关联(按 T4.1 决策每用户一个购物车);`cart_items` 含 `cart_id`(带索引)、`product_id` NOT NULL、可空 `variant_id`(可售单元规则)、`quantity` 与加入时快照 `price_cents`;同商品+变体行合并由服务层保证(部分唯一索引非三库可移植);up/down 往返已验证(每次 rollback 一步)
- [x] T4.3 加购/改数量/删除/清空 action 与页面 —— 2026-09-29 完成:`services/cart_service.go`(购物车懒创建,同商品+变体行合并,加购与改量都有库存上限,经购物车关联校验所有权,变体/商品可售单元解析,加购时价格快照);`/cart` 页面(行含图片/变体/行小计,改量,移除,清空,合计,结算占位)与商品详情页加购表单(数量输入,变体单选在表单内,无可售项时按钮禁用;游客按 T4.1 弹回登录)。环境门控集成测试覆盖合并/上限/所有权/变体流程;E2E 已验证完整页面流程
- [ ] T4.4 购物车页:金额小计、库存与下架校验(展示不可购买状态)
- [x] T4.5 服务层:`CartService`(校验、合计计算),供结算复用 —— 2026-09-29 完成:购物车逻辑全部收敛在 `services/cart_service.go`(购物车懒创建、行合并、库存上限、经购物车关联的所有权校验、可售单元解析、价格快照、只计可购行的小计与缺货标记);action 只做参数绑定与重定向编排;购物车行改为单查询批量加载商品;M5 结算将直接复用 `CartLines`。环境门控集成测试覆盖(懒创建幂等、合并、上限、所有权、变体、库存下跌状态)

## M5 订单与结算

前置依赖:M3、M4。

- [x] T5.1 迁移:`orders`(单号、用户、收货地址快照、金额小计/优惠/运费/实付、状态、支付方式)+ `order_items`(商品快照:名称、图、单价、数量)—— 2026-09-29 完成:`orders` 含 `order_no` UNIQUE、带索引 `user_id`、扁平化地址快照(收件人/电话/单行地址)、`subtotal_cents`/`discount_cents`(M7 备用)/`shipping_cents`/`total_cents`、`status`(pending/paid/shipped/completed/cancelled/refunded,默认 pending)、`payment_method`(支付前为空,M6 写入);`order_items` 快照商品/变体名称、图片 key、`unit_price_cents`,可空 `variant_id`;行小计可推导不入库;up/down 往返已验证
- [x] T5.2 结算页:选地址 → 选支付方式 → 提交订单(金额由服务端重算,不信任前端)—— 2026-09-29 完成:`services/order_service.go` 的 `PlaceOrder`(校验购物车行全部可购,快照本人地址,按行快照服务端重算金额,生成 `SO<UTC时间戳>-<随机>` 单号,订单 + 明细 + 清购物车在 `repo.WithTx` 单事务完成);`/checkout` 页面(地址单选默认选中、商品摘要、支付单选、合计);最小 `/orders/<order-no>` 详情页(T5.5 再扩展账户区视图)。错误:空购物车、不可购行、缺支付方式、他人地址。环境门控集成测试 + 完整 E2E(结算渲染、下单、详情渲染、购物车清空、跨用户 404)
- [x] T5.3 订单状态机:`pending → paid → shipped → completed`,及 `cancelled / refunded`;状态流转集中在服务层 —— 2026-09-29 完成:`services/order_state_service.go` 持有状态常量、合法流转表与 `TransitionOrder`(状态列的唯一写入方);终态拒绝再流转,未知目标状态报可读错误。环境门控集成测试覆盖(完整链路、非法跳转、终态拒绝、订单不存在)
- [ ] T5.4 ❓ 下单扣库存策略(下单预扣 + 取消回补,还是支付后扣),定了写在状态机旁
- [ ] T5.5 用户中心:订单列表(按状态筛选)、订单详情
- [ ] T5.6 取消订单(仅 pending)、订单超时未支付自动取消(定时任务)

## M6 支付

前置依赖:M5。

- [ ] T6.1 支付网关抽象:`PaymentGateway` 接口 + 网关注册表
- [ ] T6.2 假支付网关(本地/测试一键标记已支付),保证全流程可测
- [ ] T6.3 ❓ 真实渠道:支付宝 / 微信支付 / Stripe,按目标市场选型后再拆任务
- [ ] T6.4 支付回调:验签、幂等、更新订单状态
- [ ] T6.5 支付页/收银台跳转与结果页(成功/失败)

## M7 优惠券

前置依赖:M5(金额计算已集中在服务层)。

- [ ] T7.1 迁移:`coupons`(类型:满减/折扣、门槛、面值、有效期、总量)+ `coupon_redemptions`(用户×订单核销记录,防重复用)
- [ ] T7.2 admin 券管理:创建、列表、停用
- [ ] T7.3 结算时选券/输码:金额重算集成进订单服务
- [ ] T7.4 ❓ 取消/退款时券回退策略
- [ ] T7.5 (可选)用户领券中心

## M8 物流追踪

前置依赖:M5(发货挂在订单上)。

- [ ] T8.1 迁移:`shipments`(订单、承运商、运单号、状态、轨迹 JSON)
- [ ] T8.2 admin 发货:录入承运商+单号,订单转 `shipped`
- [ ] T8.3 ❓ 轨迹来源:对接查询服务(快递 100 / AfterShip 等,按可用性选型)或先做手动录入节点;定好后补任务
- [ ] T8.4 用户中心:订单物流页(时间线展示轨迹)

## M9 管理后台整合

前置依赖:M1–M8 各自的 admin 页面陆续就位。

- [ ] T9.1 ❓ 管理员体系:独立 `admin_users` 表,还是 users 加角色字段;后台登录与鉴权中间件
- [ ] T9.2 仪表盘:今日/近期订单数、销售额、待处理事项(待发货、待退款)
- [ ] T9.3 订单管理:列表(状态筛选、搜索)、详情、发货入口、退款操作
- [ ] T9.4 各模块 admin 页面体验统一(分页、筛选、空状态、确认弹窗)
- [ ] T9.5 ❓ 需要的话加操作审计 `admin_logs`

## M10 打磨与发布

前置依赖:全部。

- [ ] T10.1 种子数据:演示分类/商品/管理员账号(`go run . repl` 或 seed 命令)
- [ ] T10.2 全流程回归:注册 → 浏览 → 加购 → 结算 → 支付(假网关)→ 后台发货 → 前台查物流 → 完成
- [ ] T10.3 错误处理与日志审查(5xx 页面、关键路径日志)
- [ ] T10.4 `go test ./...` 全绿,补齐核心服务层测试(金额计算、状态机)
- [ ] T10.5 桌面端打包验证:`airway desktop:init` → `wails3 task dev` 跑通
- [ ] T10.6 README 功能清单更新为已完成状态

---

## 暂不排期(Phase 2/3,待规划)

- AirShop 特有功能(阶段二,核心系统完成后规划)
- AI 相关功能(阶段三,待规划)

## 进度记录

| 里程碑 | 完成日期 | 备注 |
|---|---|---|
| | | |
