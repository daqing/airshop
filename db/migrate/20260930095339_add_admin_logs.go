package migrations

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260930095339", "add_admin_logs", func(m *schema.Migrator) {
		m.CreateTable("admin_logs", func(t *schema.Table) {
			t.ID()
			t.BigInt("admin_id")
			t.String("action", 64)
			t.String("entity", 64)
			t.String("entity_id", 64)
			t.String("detail", 512)
			t.Timestamps()
			t.Index("admin_id")
			t.Index("created_at")
		})
	})
}
