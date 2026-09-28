# AirShop 编码与表结构约定

所有迁移和模型共享的字段约定,在 T0.2 中确定。约定跟随 Airway v0.18 脚手架
的行为——框架已经规定的地方直接采用,不另起炉灶。
英文版见 [CONVENTIONS.md](CONVENTIONS.md),两份内容一一对应。

## 主键

- 每张表只有一个代理主键 `id`,自增整数:Postgres 用 `BIGSERIAL PRIMARY KEY`,
  MySQL 用 `BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY`,SQLite 用
  `INTEGER PRIMARY KEY AUTOINCREMENT`(即 `airway generate model` 按方言
  生成的原样)。
- Go 侧统一使用 `airwaysql.IdType`(`int64`),URL 和 OpenAPI 路径参数中的
  id 都是整数。不使用 UUID 主键。

## 时间戳

- 每张表固定以下面两列结尾,与脚手架生成的一致:

  ```sql
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
  ```

- 每个模型带 `CreatedAt time.Time` 和 `UpdatedAt time.Time`,tag 为
  `db:"created_at"` / `db:"updated_at"` 和 `json:"created_at"` /
  `json:"updated_at"`。

## updated_at 的维护

repo 层不会自动维护 `updated_at`,脚手架生成的 `UpdateAction` 也不设置它
——放任不管这列就会失效。因此:

- 每次调用 `repo.UpdateByID` 都必须显式带上 `"updated_at": time.Now()`。
- 更新逻辑尽量收敛到 service 层,让这条规则在每个聚合内只落在一个地方,
  而不是散落在每个 action。
- 为真实资源生成脚手架后,发布前要给生成的 `UpdateAction` 补上
  `updated_at` 键。

## 命名

- 表名为复数 snake_case(`products`、`order_items`),与生成模型中
  `TableName()` 的返回值一致。
- 列名 snake_case。外键列命名为 `<entity>_id`(`user_id`、`order_id`),
  类型 `BIGINT`。

## 删除

- **默认硬删除。** `repo.DeleteByID` 直接删行;初期任何表都不加软删除列。
- 通常靠软删除覆盖的业务需求,改用以下机制:
  - 状态字段——例如商品用"下架"(`active → inactive`)而不是删除;
  - 订单状态机——订单永不删除;
  - 快照——`order_items` 在下单时复制商品名称/图片/价格,删除商品不会
    破坏订单历史。
- 将来某张表确实需要软删除时,只给该表加可空的 `deleted_at TIMESTAMP`
  列,并在触碰该表的每个查询里手工过滤 `deleted_at IS NULL`。软删除
  按表逐个启用,永不全局启用。

## 外键

- **迁移中不声明数据库级 `FOREIGN KEY` 约束。** 关联就是裸的
  `BIGINT <entity>_id` 列;引用完整性由服务层保证。
- 理由:与脚手架一致(它不生成外键)、迁移在 Postgres / MySQL / SQLite
  之间可移植、up/down 迁移不用处理依赖顺序。

## 金额与库存

- **金额一律用 `BIGINT` 存最小货币单位**——美元为美分,人民币为分。列名带
  显式 `_cents` 后缀(`price_cents`、`subtotal_cents`、`discount_cents`、
  `shipping_fee_cents`、`total_cents`);Go 侧类型 `int64`。不用浮点,
  也不用 `NUMERIC`/`DECIMAL`:整数最小单位精确无误差、三种数据库行为一致,
  在 SQL 和 Go 里做加减比较都安全。
- **单一门店货币。** 货币是全店设置,不是逐行字段;将来若支持多货币
  (核心范围之外)才引入逐行的货币代码列。
- **金额计算只做一次,全程 int64。** 折扣作用于小计,最后一步四舍五入到
  整分;中间过程不取整。总额始终由服务端重算。
- **库存是 `INTEGER NOT NULL DEFAULT 0`**,不为负、不支持小数。扣减/回补
  策略在 T5.4 一并定版。
- 百分比数值(如折扣券)用普通整数 0–100;只有将来需要小于 1% 的精度时
  才升级为基点。
- 展示格式化(分 → "¥12.34")收敛在一个共享小工具里,首个 UI 需要时
  (M1)创建;禁止在调用点内联格式化。

## 商品变体

- 变体从第一天就支持(T1.3,2026-09-28 定)。商品在 `product_variants` 里
  有至少一行启用的变体时,**通过变体销售**;否则商品行本身即可售单元
  (其 `price_cents`/`stock` 就是默认价与默认库存)。
- 可售单元的判定**只在 Product 服务层**(T1.4);购物车、结算、后台
  一律不直接查 `product_variants`。
- 每个变体即可售 SKU:`product_id`、组合的展示名 `name`(如 "红色 / M")、
  独立 `price_cents`、独立 `stock`、`active`、`sort_order`。结构化多轴
  选项(options/option-values 表族)刻意后置;将来引入时只负责给变体
  打标签,不改变价格库存的读取路径。
- 已被购物车或订单引用的变体永不删除:置 `active = FALSE`,保住历史。

## 决策记录

- 2026-09-28 —— T0.2 由 David Zhang 拍板:硬删除 + 状态字段替代,
  不做数据库级外键,显式维护 `updated_at`。
- 2026-09-28 —— T0.5 由 David Zhang 拍板:金额用 BIGINT 最小货币单位
  (`_cents` 后缀,Go 侧 int64),单一门店货币,库存为非负 INTEGER,
  百分比为整数 0–100。
- 2026-09-28 —— T1.3 由 David Zhang 拍板:现在就支持变体
  (`product_variants` 表);可售单元判定收在 Product 服务层;结构化
  选项轴后置;被引用的变体停用而非删除。
