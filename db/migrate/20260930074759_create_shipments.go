package migrations

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260930074759", "create_shipments", func(m *schema.Migrator) {
		m.CreateTable("shipments", func(t *schema.Table) {
			t.ID()
			t.BigInt("order_id")
			t.String("carrier", 64)
			t.String("tracking_no", 64)
			t.String("status", 20)
			t.JSON("events")
			t.Timestamps()
			t.Index("order_id")
		})
	})
}
