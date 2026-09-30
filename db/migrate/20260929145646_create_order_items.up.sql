CREATE TABLE order_items (
	id BIGSERIAL PRIMARY KEY,
	order_id BIGINT NOT NULL,
	product_id BIGINT NOT NULL,
	variant_id BIGINT,
	product_name VARCHAR(255) NOT NULL,
	variant_name VARCHAR(255) NOT NULL DEFAULT '',
	image_key VARCHAR(255) NOT NULL DEFAULT '',
	unit_price_cents BIGINT NOT NULL,
	quantity INTEGER NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);
