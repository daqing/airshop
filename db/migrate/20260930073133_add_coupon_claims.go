package migrations

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260930073133", "add_coupon_claims", func(m *schema.Migrator) {
		m.CreateTable("coupon_claims", func(t *schema.Table) {
			t.ID()
			t.BigInt("coupon_id")
			t.BigInt("user_id")
			t.Timestamps()
			t.UniqueIndex("coupon_id", "user_id")
			t.Index("user_id")
		})
	})
}
