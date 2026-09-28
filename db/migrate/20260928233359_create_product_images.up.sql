CREATE TABLE product_images (
	id BIGSERIAL PRIMARY KEY,
	product_id BIGINT NOT NULL,
	key VARCHAR(255) NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_product_images_product_id ON product_images (product_id);
