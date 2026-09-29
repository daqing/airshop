package services

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/models"
)

var addressPhonePattern = regexp.MustCompile(`^\d{5,20}$`)

var (
	ErrAddressRecipientRequired = errors.New("recipient name is required")
	ErrAddressPhoneInvalid      = errors.New("enter a valid contact phone number")
	ErrAddressStreetRequired    = errors.New("street address is required")
	ErrAddressNotFound          = errors.New("address not found")
)

type AddressInput struct {
	Recipient string
	Phone     string
	Province  string
	City      string
	District  string
	Street    string
	IsDefault bool
}

func (in AddressInput) validate() error {
	if strings.TrimSpace(in.Recipient) == "" {
		return ErrAddressRecipientRequired
	}
	if !addressPhonePattern.MatchString(strings.ReplaceAll(strings.TrimSpace(in.Phone), "-", "")) {
		return ErrAddressPhoneInvalid
	}
	if strings.TrimSpace(in.Street) == "" {
		return ErrAddressStreetRequired
	}
	return nil
}

// ListAddresses returns the user's addresses, default first then newest.
func ListAddresses(userID int64) ([]*models.Address, error) {
	b := sql.Select("*").
		From("addresses").
		Where(sql.Eq("user_id", userID)).
		OrderBy("is_default DESC, id DESC")
	return repo.Find[models.Address](repo.CurrentDB(), b)
}

func findOwnedAddress(userID, addressID int64) (*models.Address, error) {
	addr, err := repo.FindByID[models.Address](sql.IdType(addressID))
	if err != nil {
		return nil, err
	}
	if addr == nil || addr.UserID != userID {
		return nil, ErrAddressNotFound
	}
	return addr, nil
}

// FindAddress returns the user's owned address, or ErrAddressNotFound when
// it does not exist or belongs to someone else.
func FindAddress(userID, addressID int64) (*models.Address, error) {
	return findOwnedAddress(userID, addressID)
}

// CreateAddress adds an address for the user. The first address becomes the
// default automatically; an explicit default clears the previous one.
func CreateAddress(userID int64, in AddressInput) (*models.Address, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}

	existing, err := repo.FindBy[models.Address](sql.H{"user_id": userID})
	if err != nil {
		return nil, err
	}
	isDefault := in.IsDefault || len(existing) == 0

	if isDefault && len(existing) > 0 {
		if err := clearDefaultAddress(userID); err != nil {
			return nil, err
		}
	}

	return repo.CreateFrom[models.Address](sql.H{
		"user_id":    userID,
		"recipient":  strings.TrimSpace(in.Recipient),
		"phone":      strings.TrimSpace(in.Phone),
		"province":   strings.TrimSpace(in.Province),
		"city":       strings.TrimSpace(in.City),
		"district":   strings.TrimSpace(in.District),
		"street":     strings.TrimSpace(in.Street),
		"is_default": isDefault,
	})
}

// UpdateAddress modifies an owned address; ownership mismatches read as
// not-found.
func UpdateAddress(userID, addressID int64, in AddressInput) error {
	if _, err := findOwnedAddress(userID, addressID); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}

	if in.IsDefault {
		if err := clearDefaultAddressExcept(userID, addressID); err != nil {
			return err
		}
	}

	return repo.UpdateByID[models.Address](sql.IdType(addressID), sql.H{
		"recipient":  strings.TrimSpace(in.Recipient),
		"phone":      strings.TrimSpace(in.Phone),
		"province":   strings.TrimSpace(in.Province),
		"city":       strings.TrimSpace(in.City),
		"district":   strings.TrimSpace(in.District),
		"street":     strings.TrimSpace(in.Street),
		"is_default": in.IsDefault,
		"updated_at": time.Now().UTC(),
	})
}

// DeleteAddress removes an owned address. When the default address is
// deleted, the newest remaining one becomes the default.
func DeleteAddress(userID, addressID int64) error {
	addr, err := findOwnedAddress(userID, addressID)
	if err != nil {
		return err
	}

	if err := repo.DeleteByID[models.Address](sql.IdType(addressID)); err != nil {
		return err
	}

	if !addr.IsDefault {
		return nil
	}

	remaining, err := repo.FindBy[models.Address](sql.H{"user_id": userID})
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return nil
	}

	promote := remaining[0]
	for _, a := range remaining[1:] {
		if a.ID > promote.ID {
			promote = a
		}
	}
	return repo.UpdateByID[models.Address](promote.ID, sql.H{
		"is_default": true,
		"updated_at": time.Now().UTC(),
	})
}

// SetDefaultAddress makes the address the only default for the user.
func SetDefaultAddress(userID, addressID int64) error {
	if _, err := findOwnedAddress(userID, addressID); err != nil {
		return err
	}

	if err := clearDefaultAddress(userID); err != nil {
		return err
	}

	return repo.UpdateByID[models.Address](sql.IdType(addressID), sql.H{
		"is_default": true,
		"updated_at": time.Now().UTC(),
	})
}

func clearDefaultAddress(userID int64) error {
	return repo.UpdateWhere[models.Address](sql.H{"is_default": false}, sql.Eq("user_id", userID))
}

func clearDefaultAddressExcept(userID, addressID int64) error {
	return repo.UpdateWhere[models.Address](
		sql.H{"is_default": false},
		sql.AllOf(sql.Eq("user_id", userID), sql.NotEq("id", addressID)),
	)
}
