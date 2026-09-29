CREATE TABLE orders (
	id BIGSERIAL PRIMARY KEY,
	order_no VARCHAR(32) NOT NULL UNIQUE,
	user_id BIGINT NOT NULL,
	ship_recipient VARCHAR(255) NOT NULL,
	ship_phone VARCHAR(20) NOT NULL,
	ship_address VARCHAR(512) NOT NULL,
	subtotal_cents BIGINT NOT NULL,
	discount_cents BIGINT NOT NULL DEFAULT 0,
	shipping_cents BIGINT NOT NULL DEFAULT 0,
	total_cents BIGINT NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'pending',
	payment_method VARCHAR(30) NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_user_id ON orders (user_id);
