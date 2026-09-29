CREATE TABLE coupon_redemptions (
	id BIGSERIAL PRIMARY KEY,
	coupon_id BIGINT NOT NULL,
	user_id BIGINT NOT NULL,
	order_id BIGINT NOT NULL UNIQUE,
	discount_cents BIGINT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_coupon_redemptions_coupon_id ON coupon_redemptions (coupon_id);
CREATE INDEX idx_coupon_redemptions_user_id ON coupon_redemptions (user_id);
