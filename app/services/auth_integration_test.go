package services

import (
	"os"
	"testing"
	"time"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestAuthServiceFlow exercises the full service layer end to end. It runs
// only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestAuthServiceFlow
func TestAuthServiceFlow(t *testing.T) {
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
		_ = repo.DeleteWhere[models.Session](sql.H{})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": "13900009999"})
	})

	phone := "13900009999"

	// The code path: send, then consume the stored code.
	if _, err := SendSignInCode(phone); err != nil {
		t.Fatalf("send code: %v", err)
	}
	codeStoreMu.Lock()
	entry := codeStore[phone]
	codeStoreMu.Unlock()
	if err := ConsumeSignInCode(phone, entry.code); err != nil {
		t.Fatalf("consume code: %v", err)
	}

	// First sign-in creates the account.
	user, err := SignInOrSignUp(phone)
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	if user.PhoneNumber != phone {
		t.Fatalf("expected phone %q, got %q", phone, user.PhoneNumber)
	}

	// Second sign-in reuses the account.
	again, err := SignInOrSignUp(phone)
	if err != nil {
		t.Fatalf("second sign in: %v", err)
	}
	if again.ID != user.ID {
		t.Fatalf("expected the same user on re-sign-in, got %d vs %d", again.ID, user.ID)
	}

	// Session lifecycle.
	token, err := CreateSession(int64(user.ID))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	resolved, err := SessionUser(token)
	if err != nil {
		t.Fatalf("session user: %v", err)
	}
	if resolved == nil || resolved.ID != user.ID {
		t.Fatalf("expected the session to resolve to user %d, got %#v", user.ID, resolved)
	}

	if err := DestroySession(token); err != nil {
		t.Fatalf("destroy session: %v", err)
	}
	if resolved, _ = SessionUser(token); resolved != nil {
		t.Fatal("expected the session to be gone after sign-out")
	}

	// Expired sessions self-clean on lookup.
	expiredToken, err := CreateSession(int64(user.ID))
	if err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	session, err := repo.FindOneBy[models.Session](sql.H{"token": expiredToken})
	if err != nil || session == nil {
		t.Fatalf("find fresh session: %v", err)
	}
	if err := repo.UpdateByID[models.Session](session.ID, sql.H{
		"expires_at": time.Now().UTC().Add(-time.Hour),
		"updated_at": time.Now().UTC(),
	}); err != nil {
		t.Fatalf("expire session: %v", err)
	}
	if resolved, _ = SessionUser(expiredToken); resolved != nil {
		t.Fatal("expected an expired session to resolve to no user")
	}
	if check, _ := repo.FindOneBy[models.Session](sql.H{"token": expiredToken}); check != nil {
		t.Fatal("expected the expired session row to be deleted on lookup")
	}
}
