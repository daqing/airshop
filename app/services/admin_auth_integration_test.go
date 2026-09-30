package services

import (
	"os"
	"testing"
	"time"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"
	"golang.org/x/crypto/bcrypt"

	"github.com/daqing/airshop/app/models"
)

// TestAdminAuth runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestAdminAuth
func TestAdminAuth(t *testing.T) {
	dsn := os.Getenv("AIRWAY_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("set AIRWAY_PG_TEST_DSN to run this integration test")
	}

	if err := migrate.Run(migrate.Options{
		DSN:          dsn,
		Migrations:   os.DirFS("../../db/migrate"),
		SnapshotPath: "",
	}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := repo.SetupDB(dsn); err != nil {
		t.Fatalf("setup db: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.AdminSession](sql.H{})
		_ = repo.DeleteWhere[models.AdminUser](sql.H{})
	})

	// Login with wrong credentials fails without creating a session.
	if _, err := AdminLogin("root", "nope"); err != ErrAdminLoginFailed {
		t.Fatalf("expected login failure, got %v", err)
	}

	// Create an admin directly and sign in.
	hash, err := bcrypt.GenerateFromPassword([]byte("s3cret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	admin, err := repo.CreateFrom[models.AdminUser](sql.H{
		"username": "root", "password_hash": string(hash), "display_name": "Root", "status": AdminStatusActive,
	})
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	token, err := AdminLogin("root", "s3cret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	resolved, err := AdminSessionAdmin(token)
	if err != nil || resolved == nil || resolved.ID != admin.ID {
		t.Fatalf("expected the session to resolve to admin %d, got %#v / %v", admin.ID, resolved, err)
	}

	// Disabled admins do not resolve.
	if err := repo.UpdateByID[models.AdminUser](admin.ID, sql.H{"status": AdminStatusDisabled, "updated_at": time.Now().UTC()}); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if resolved, _ = AdminSessionAdmin(token); resolved != nil {
		t.Fatal("expected a disabled admin to stop resolving")
	}

	// Sign-out destroys the session.
	if err := DestroyAdminSession(token); err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if resolved, _ = AdminSessionAdmin(token); resolved != nil {
		t.Fatal("expected the session to be gone after sign-out")
	}
	if _, err := AdminLogin("root", "s3cret"); err != ErrAdminDisabled {
		t.Fatalf("expected disabled login to fail, got %v", err)
	}
}
