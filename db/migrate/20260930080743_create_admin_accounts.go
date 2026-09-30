package migrations

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20260930080743", "create_admin_accounts", func(m *schema.Migrator) {
		m.CreateTable("admin_users", func(t *schema.Table) {
			t.ID()
			t.String("username", 64)
			t.String("password_hash", 255)
			t.String("display_name", 255)
			t.String("status", 20)
			t.Timestamps()
			t.Index("username").Unique()
		})

		m.CreateTable("admin_sessions", func(t *schema.Table) {
			t.ID()
			t.String("token", 64)
			t.BigInt("admin_id")
			t.DateTime("expires_at")
			t.Timestamps()
			t.Index("token").Unique()
			t.Index("admin_id")
		})
	})
}
