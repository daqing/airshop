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
- [x] T5.4 ❓ 下单扣库存策略 —— 2026-09-29 由 David Zhang 拍板:**支付后扣减**。`PlaceOrder` 不动库存;扣减挂在状态机的 `paid` 流转上(支付回调与假网关的唯一路径),退款自动回补数量,`pending → cancelled` 不涉及库存(尚未扣过)。扣减以零为下限,且与状态变更同事务完成。环境门控集成测试覆盖(`TestOrderStockFollowsPayment`:商品行与变体行贯穿 支付/退款/取消)
- [x] T5.5 用户中心:订单列表(按状态筛选)、订单详情 —— 2026-09-29 完成:`services.ListOrders`(状态筛选 + 每页 10 条 + 计数)与 `/orders` 列表页(状态 chip、最新在前、状态徽标、金额、Prev/Next 分页),账户页已挂 My orders 入口;`/orders/<order-no>` 详情页沿用 T5.2 所建。所有权全域校验(他人单号 404)。E2E 已验证(全部/按状态列表、徽标、详情链接)
- [x] T5.6 取消订单(仅 pending)、订单超时未支付自动取消(定时任务)—— 2026-09-29 完成:`services.CancelOrder`(经单号查找即所有权校验,状态机强制仅 pending 可取消),订单详情页与列表页都有取消按钮;`services.CancelExpiredOrders` 取消超过 30 分钟的 pending 订单,由 main 启动的后台清扫器执行(启动即扫一次,此后每分钟一次)。过期阈值用本地墙钟计算以匹配数据库 `CURRENT_TIMESTAMP` 默认值(已在 CONVENTIONS 的 UTC 规则旁注明)。环境门控集成测试(本人/陌生人取消、paid 拒绝、清扫器作用域)+ E2E(详情页取消流程、回填订单的开机清扫)

## M6 支付

前置依赖:M5。

- [x] T6.1 支付网关抽象:`PaymentGateway` 接口 + 网关注册表 —— 2026-09-29 完成:`services/payment_gateway.go` 定义最小接口(`Name()` 兼作 `orders.payment_method`,`PayLink(order)` 为买家跳转目标)与互斥锁注册表(`RegisterGateway` 供 init 调用、`Gateway`、排序的 `GatewayNames`);重复注册即替换。支付确认路由保持各网关自持、暂不纳入接口——真实渠道的回调形态明确后再扩展(T6.3+)。单测覆盖(注册、名称排序、查找、替换)
- [x] T6.2 假支付网关(本地/测试一键标记已支付),保证全流程可测 —— 2026-09-29 完成:`services/payment_fake.go` 注册 `fake` 网关,PayLink 指向内部收银台 `/pay/fake?order=<no>`;收银台页(订单总额、状态感知的 Pay now 按钮)经 `TransitionOrder(pending → paid)` 确认支付,同步触发 T5.4 的扣库存并拒绝重复支付;结算页支付单选改为从 `GatewayNames()` 动态渲染,`PlaceOrder` 校验支付方式必须在注册表中。PlaceOrder 集成测试(未知网关拒绝)+ 完整支付闭环 E2E(收银台渲染、支付、库存 5→3、重复支付拒绝)
- [x] T6.3 ❓ 真实渠道:支付宝 / 微信支付 / Stripe —— 2026-09-29 由 David Zhang 拍板:**微信支付与支付宝;暂不支持 Stripe**。`PaymentGateway` 注册表保持扩展缝:新渠道实现接口并在 `init()` 自注册,建议以配置齐全为注册门槛,让结算页只列出可用渠道。实际接入等商户资质就绪——见 T6.3a 与 T6.3b。
- [ ] T6.3a 微信支付接入(待商户资质:商户号、appid、APIv3 密钥、证书序列号与私钥)——注册 `wechat` 网关(以环境变量齐全为门槛),实现统一下单调用与验签异步回调,回调驱动 `TransitionOrder(paid)`
- [ ] T6.3b 支付宝接入(待商户资质:应用 appid、应用私钥、支付宝公钥)——注册 `alipay` 网关(以环境变量齐全为门槛),实现交易创建调用与验签异步回调,回调驱动 `TransitionOrder(paid)`
- [x] T6.4 支付回调:验签、幂等、更新订单状态 —— 2026-09-29 完成:`services.MarkOrderPaid` 是所有网关共用的支付成功入口(今天是假收银台;T6.3a/b 的验签异步回调按各自 SDK 验签后同样调它)。幂等契约:已支付订单的重复通知静默成功(网关停止重试)、库存只扣一次;退款/取消订单拒绝支付;未知订单以 not-found 暴露供网关记日志。验签本身按网关实现,随 T6.3a/b 落地。环境门控集成测试覆盖(扣一次、幂等重复、退款拒绝、not-found);E2E 验证二次确认静默落回订单页且库存不重复扣
- [x] T6.5 支付页/收银台跳转与结果页(成功/失败)—— 2026-09-29 完成:T6.2 的流程(结算 → 网关 PayLink → 收银台 → 确认)落地到订单详情页,以 `paid` 标记驱动绿色"Payment received"横幅——仅在订单确为 paid 时渲染(伪造 pending 订单的查询参数不显示);失败路径回到收银台带错误横幅,PlaceOrder 失败在结算页呈现。E2E 覆盖(真实回跳显示横幅、无标记不显示、pending 伪造不显示)

## M7 优惠券

前置依赖:M5(金额计算已集中在服务层)。

- [x] T7.1 迁移:`coupons`(类型:满减/折扣、门槛、面值、有效期、总量)+ `coupon_redemptions`(用户×订单核销记录,防重复用)—— 2026-09-29 完成:`coupons` 含唯一 `code`(结算可输码)、`type` fixed/percent 且按 T0.5 规则分列 `value_cents` 与 `percent_off`、门槛 `threshold_cents`、`total_count`(0 = 不限,用量由核销计数推导)、可空 `starts_at`/`expires_at` 窗口与 `enabled` 开关;`coupon_redemptions` 快照实付优惠 `discount_cents`,`order_id` UNIQUE 作数据库兜底(一单一张),另加券/用户索引——同人同券防重复由服务层保证(T7.3,组合部分索引非三库可移植);up/down 往返已验证
- [x] T7.2 admin 券管理:创建、列表、停用 —— 2026-09-29 完成:`services/coupon_service.go`(码格式/唯一校验,留空自动生成,类型化校验——fixed 需正数金额、percent 需 1-100 且不填金额,时间窗顺序,`updated_at` 用 UTC),用量由核销记录计数;`/admin/coupons` 列表(优惠展示、门槛、用量 n/上限或 ∞、时间窗、启用/停用开关)与创建/编辑表单(datetime-local 时间窗、留空码自动生成)。E2E 已验证(percent 自动码、fixed 指定码、重复码与类型错配 422、编辑回填/更新、停用/启用)
- [x] T7.3 结算时选券/输码:金额重算集成进订单服务 —— 2026-09-29 完成:`services.ApplicableCoupon` / `CouponDiscountForLoaded` / `ApplicableCouponsFor`(启用、时间窗、同人限用一次、总量上限、门槛;fixed 面值封顶至小计、percent 按 T0.5 一次性四舍五入);`PlaceOrder` 接收券码,在订单事务内以券行 `FOR UPDATE` 锁定后重算折扣(并发结算不会突破总量上限)并写入核销行;结算页列出可用券提示并接受大小写不敏感的券码输入。环境门控集成测试覆盖(折扣计算、门槛/复用/耗尽、核销快照、无券订单)与 E2E(提示列表、小写券码、订单折扣、复用拒绝)
- [x] T7.4 ❓ 取消/退款时券回退策略 —— 2026-09-29 由 David Zhang 拍板:**券归还用户**——取消 pending 订单或退款 paid 订单时,在流转事务内删除该订单的核销行,单次券恢复可用(与库存退款回补同一原则:交易未成,消耗即归还)。环境门控集成测试覆盖(取消后恢复、paid 期间已消耗、退款后恢复、核销行清零)
- [x] T7.5 (可选)用户领券中心 —— 2026-09-30 完成:券改为"先领取、后使用"——`coupon_claims` 表(Go DSL 迁移,券×用户唯一)与 `services.ClaimCoupon`(启用/时间窗/领取池校验);结算可用性要求"已领取且未核销";`/coupons` 领券中心页列出可领券、我的券(含已用徽标),账户页挂入口。`total_count` 现在限制领取数(核销数 ≤ 领取数由构造保证)。既有门控测试改为先领取后用券,新测试与 E2E 覆盖(领取 → 中心标记我的 → 结算生效 → 已用徽标)

## M8 物流追踪

前置依赖:M5(发货挂在订单上)。

- [x] T8.1 迁移:`shipments`(订单、承运商、运单号、状态、轨迹 JSON)—— 2026-09-29 完成:首个使用 v0.19.1 Go DSL 的迁移(`m.CreateTable`,down 自动反转);`order_id` 带索引,`carrier`/`tracking_no` VARCHAR(64),`status` VARCHAR(20) 默认 `created`(由服务层约束),轨迹 `events` 为 JSONB(服务层追加 `{time, description}` 数组)。up/down 往返已验证
- [x] T8.2 admin 发货:录入承运商+单号,订单转 `shipped` —— 2026-09-29 完成:`services/shipment_service.go`(`ShipOrder` 校验承运商/单号、仅 paid 可发货、一单一运单并写入首条 "Label created" 事件、订单转 shipped 且流转失败回滚运单行;`UpdateShipmentStatus` 沿 created → in_transit → delivered 追加事件);`/admin/shipments` 页面(待发货 paid 订单内联发货表单、已发货表带 In transit/Delivered 按钮)。环境门控集成测试(输入校验、pending 拒绝、含事件的正常路径、重复发货拒绝、状态生命周期)与 E2E 覆盖
- [x] T8.3 ❓ 轨迹来源 —— 2026-09-29 由 David Zhang 拍板:**先手动录入**。admin 可按运单追加轨迹事件(`POST /admin/shipments/:id/events`,存入 events JSON 轨迹);对接轨迹服务(快递 100 / AfterShip / 承运商 webhook)推迟到有真实运量时,届时只替换事件的来源方式——前台时间线读同一条轨迹。(T8.4 的前台时间线随本任务一并落地。)
- [x] T8.4 用户中心:订单物流页(时间线展示轨迹)—— 2026-09-29 完成:订单详情页在有运单时显示 Tracking 盒子(承运商 + 单号、运单状态、事件时间线最新在前),解析与展示都在订单所有权校验内。E2E 已验证(admin 发货 + 手动事件 + 状态推进,用户时间线渲染)

## M9 管理后台整合

前置依赖:M1–M8 各自的 admin 页面陆续就位。

- [x] T9.1 ❓ 管理员体系:独立 `admin_users` 表,还是 users 加角色字段;后台登录与鉴权中间件 —— 2026-09-29 由 David Zhang 拍板:**独立 `admin_users` + `admin_sessions` 表,独立用户名/密码登录(bcrypt)**,登录页 `/admin/login`——顾客短信登录与后台登录完全隔离。启动时引导首个管理员(`ADMIN_USERNAME`/`ADMIN_PASSWORD`,本地默认 admin/admin123 并打警告);`LoadAdminUser` + `RequireAdmin` 守卫登录对以外的全部 `/admin` 路由;后台布局显示当前管理员与登出按钮。环境门控集成测试(错误凭据、登录、会话解析、禁用拒绝、登出)与 E2E(访客弹登录、错误密码、后台访问、登出)覆盖
- [x] T9.2 仪表盘:今日/近期订单数、销售额、待处理事项(待发货、待退款)—— 2026-09-29 完成:`services.DashboardData`(今日/近 7 天的订单数与销售额,统计口径为 paid+shipped+completed,时间阈值按 CURRENT_TIMESTAMP 规则用本地墙钟;按状态的待办计数),仪表盘页重写为统计卡片 + 待办卡片(链到发货/订单页) + 模块快捷入口。环境门控集成测试(今日/本周窗口、pending 与 refunded 不计、ToShip 不看时效)与 E2E(支付后数字联动)覆盖
- [x] T9.3 订单管理:列表(状态筛选、搜索)、详情、发货入口、退款操作 —— 2026-09-29 完成:`services.AdminListOrders`(状态筛选、单号 ILIKE 搜索、分页)/`AdminFindOrderByNo`/`AdminRefundOrder`(状态机驱动库存与券回补);`/admin/orders` 列表(chip、搜索、分页)与 `/admin/orders/<单号>` 详情(摘要、明细、收货、追踪行、可退款时显示退款按钮、paid 时显示发货快捷入口)。E2E 已验证(列表/搜索/筛选、详情、支付 → 退款且库存与券回补)
- [x] T9.4 各模块 admin 页面体验统一(分页、筛选、空状态、确认弹窗)—— 2026-09-29 完成:admin 布局持有共享的 `a-*` 组件样式(表格、按钮含 ghost/danger 变体、徽标、横幅/错误、分页、空态、表单、盒子、键值行、迷你按钮)与明暗变量;各 admin 页面删除重复样式块、统一改用共享类;破坏性操作(分类删除、商品图片删除)带确认对话框。layouts 测试与六个 admin 页面的 E2E 冒烟覆盖
- [x] T9.5 ❓ 需要的话加操作审计 `admin_logs` —— 2026-09-30 完成,定为 **A-轻量**:`admin_logs` 表 + 尽力而为的 `services.Audit` 助手(写入失败仅记日志、不影响原操作),埋点覆盖高风险操作——订单退款、发货、运单状态变更、券启停、商品上下架与价格变更。无 admin 查看页;用 psql 或 REPL(`services.AdminLogs`)查询。环境门控集成测试与 E2E(退款与改价的审计条目验证)覆盖

## M10 打磨与发布

前置依赖:全部。

- [x] T10.1 种子数据:演示分类/商品/管理员账号 —— 2026-09-29 完成:`services.SeedDemoDataIfEmpty` 在空目录时播种 2 个分类、5 个演示商品与 WELCOME10 折扣券;仅在 local 环境且商品表为空时于启动时执行(生产不见演示数据;管理员账号由 T9.1 的启动引导负责)。E2E 已验证(启动播种一次、前台渲染演示商品、重启不重复)
- [x] T10.2 全流程回归:注册 → 浏览 → 加购 → 结算 → 支付(假网关)→ 后台发货 → 前台查物流 → 完成 —— 2026-09-29 完成:干净库上 17 个检查点的在线回归全部通过——启动播种、管理员引导、前台浏览、短信自动注册、领券、建地址、加购、WELCOME10 结算(满 3980 减 398)、假网关支付(库存 20→18)、购物车清空、admin 发货带时间线、本人可见追踪、已送达、审计条目、仪表盘联动
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
