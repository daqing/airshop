package services

import (
	"os"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

// TestAddressService runs only when a scratch Postgres is provided:
//
//	AIRWAY_PG_TEST_DSN=postgres://...airshop_test go test ./app/services -run TestAddressService
func TestAddressService(t *testing.T) {
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

	userA, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13700000001", "status": "active"})
	if err != nil {
		t.Fatalf("seed user A: %v", err)
	}
	userB, err := repo.CreateFrom[models.User](sql.H{"phone_number": "13700000002", "status": "active"})
	if err != nil {
		t.Fatalf("seed user B: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteWhere[models.Address](sql.H{})
		_ = repo.DeleteWhere[models.User](sql.H{"phone_number": sql.Like("phone_number", "137000000%")})
	})

	in := AddressInput{Recipient: "Zhang", Phone: "13800138000", Province: "Shanghai", City: "Shanghai", Street: "No. 1 Road"}

	// The first address is forced default.
	first, err := CreateAddress(int64(userA.ID), in)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if !first.IsDefault {
		t.Fatal("expected the first address to become the default")
	}

	// A second non-default address keeps the default intact.
	second, err := CreateAddress(int64(userA.ID), AddressInput{Recipient: "Wang", Phone: "13900139000", Street: "No. 2 Road"})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if second.IsDefault {
		t.Fatal("expected the second address to stay non-default")
	}

	// Creating an explicit default clears the previous one.
	third, err := CreateAddress(int64(userA.ID), AddressInput{Recipient: "Li", Phone: "13700137000", Street: "No. 3 Road", IsDefault: true})
	if err != nil {
		t.Fatalf("create third: %v", err)
	}
	assertSingleDefault(t, int64(userA.ID), int64(third.ID))

	// Setting a default clears the others.
	if err := SetDefaultAddress(int64(userA.ID), int64(first.ID)); err != nil {
		t.Fatalf("set default: %v", err)
	}
	assertSingleDefault(t, int64(userA.ID), int64(first.ID))

	// Ownership isolation: user B cannot read, update or delete user A's address.
	if _, err := FindAddress(int64(userB.ID), int64(first.ID)); err != ErrAddressNotFound {
		t.Fatalf("expected cross-user read to be not-found, got %v", err)
	}
	if err := UpdateAddress(int64(userB.ID), int64(first.ID), in); err != ErrAddressNotFound {
		t.Fatalf("expected cross-user update to be not-found, got %v", err)
	}
	if err := DeleteAddress(int64(userB.ID), int64(first.ID)); err != ErrAddressNotFound {
		t.Fatalf("expected cross-user delete to be not-found, got %v", err)
	}
	if err := SetDefaultAddress(int64(userB.ID), int64(first.ID)); err != ErrAddressNotFound {
		t.Fatalf("expected cross-user set-default to be not-found, got %v", err)
	}

	// Deleting the default promotes the newest remaining address.
	if err := DeleteAddress(int64(userA.ID), int64(first.ID)); err != nil {
		t.Fatalf("delete default: %v", err)
	}
	assertSingleDefault(t, int64(userA.ID), int64(third.ID))

	// Validation errors.
	if _, err := CreateAddress(int64(userA.ID), AddressInput{Recipient: "", Phone: "13800138000", Street: "x"}); err != ErrAddressRecipientRequired {
		t.Fatalf("expected missing recipient to be rejected, got %v", err)
	}
	if _, err := CreateAddress(int64(userA.ID), AddressInput{Recipient: "Z", Phone: "abc", Street: "x"}); err != ErrAddressPhoneInvalid {
		t.Fatalf("expected a bad phone to be rejected, got %v", err)
	}
	if _, err := CreateAddress(int64(userA.ID), AddressInput{Recipient: "Z", Phone: "13800138000", Street: " "}); err != ErrAddressStreetRequired {
		t.Fatalf("expected a blank street to be rejected, got %v", err)
	}
}

func assertSingleDefault(t *testing.T, userID, defaultID int64) {
	t.Helper()

	list, err := ListAddresses(userID)
	if err != nil {
		t.Fatalf("list addresses: %v", err)
	}

	defaults := 0
	for _, a := range list {
		if a.IsDefault {
			defaults++
			if int64(a.ID) != defaultID {
				t.Fatalf("expected address %d to be the default, got %d", defaultID, a.ID)
			}
		}
	}
	if defaults != 1 {
		t.Fatalf("expected exactly one default among %d addresses, got %d", len(list), defaults)
	}
}
