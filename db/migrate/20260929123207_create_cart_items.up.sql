CREATE TABLE cart_items (
	id BIGSERIAL PRIMARY KEY,
	cart_id BIGINT NOT NULL,
	product_id BIGINT NOT NULL,
	variant_id BIGINT,
	quantity INTEGER NOT NULL,
	price_cents BIGINT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_cart_items_cart_id ON cart_items (cart_id);
