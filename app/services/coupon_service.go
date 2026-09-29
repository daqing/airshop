package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"

	"github.com/daqing/airshop/app/models"
)

const (
	CouponTypeFixed   = "fixed"
	CouponTypePercent = "percent"
)

var (
	couponCodePattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{2,31}$`)
	ErrCouponCodeInvalid   = errors.New("code must be 3-32 letters, digits or dashes")
	ErrCouponCodeTaken     = errors.New("code is already taken")
	ErrCouponTypeInvalid   = errors.New("type must be fixed or percent")
	ErrCouponValueInvalid  = errors.New("discount value must be positive for the chosen type")
	ErrCouponWindowInvalid = errors.New("expiry must come after the start time")
	ErrCouponNotFound      = errors.New("coupon not found")
)

type CouponInput struct {
	Code           string
	Type           string
	ValueCents     int64
	PercentOff     int
	ThresholdCents int64
	TotalCount     int
	StartsAt       *time.Time
	ExpiresAt      *time.Time
	Enabled        bool
}

func (in CouponInput) normalizeCode(autoBase string) string {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if code != "" {
		return code
	}
	base := strings.ToUpper(slugifyNonAlnum.ReplaceAllString(autoBase, "-"))
	base = strings.Trim(base, "-")
	if base == "" {
		base = "CPN"
	}
	return base + "-" + strings.ToUpper(utils.RandomHex(3))
}

func (in CouponInput) validate(code string) error {
	if !couponCodePattern.MatchString(code) {
		return ErrCouponCodeInvalid
	}
	if in.Type != CouponTypeFixed && in.Type != CouponTypePercent {
		return ErrCouponTypeInvalid
	}
	switch in.Type {
	case CouponTypeFixed:
		if in.ValueCents <= 0 || in.PercentOff != 0 {
			return ErrCouponValueInvalid
		}
	case CouponTypePercent:
		if in.PercentOff < 1 || in.PercentOff > 100 || in.ValueCents != 0 {
			return ErrCouponValueInvalid
		}
	}
	if in.ThresholdCents < 0 || in.TotalCount < 0 {
		return ErrCouponValueInvalid
	}
	if in.StartsAt != nil && in.ExpiresAt != nil && in.ExpiresAt.Before(*in.StartsAt) {
		return ErrCouponWindowInvalid
	}
	return nil
}

func couponCodeTaken(code string, selfID int64) (bool, error) {
	conflict, err := repo.FindOneBy[models.Coupon](sql.H{"code": code})
	if err != nil {
		return false, err
	}
	return conflict != nil && conflict.ID != sql.IdType(selfID), nil
}

// AdminListCoupons returns every coupon, newest first. Usage counts are
// derived by the caller from coupon_redemptions.
func AdminListCoupons() ([]*models.Coupon, error) {
	b := sql.Select("*").From("coupons").OrderBy("id DESC")
	return repo.Find[models.Coupon](repo.CurrentDB(), b)
}

func FindCoupon(id sql.IdType) (*models.Coupon, error) {
	return repo.FindByID[models.Coupon](id)
}

// CouponUsage returns how many times a coupon has been redeemed.
func CouponUsage(couponID int64) (int64, error) {
	return repo.CountWhere[models.CouponRedemption](sql.H{"coupon_id": couponID})
}

func AdminCreateCoupon(in CouponInput) (*models.Coupon, error) {
	code := in.normalizeCode(in.Type)
	if err := in.validate(code); err != nil {
		return nil, err
	}
	if taken, err := couponCodeTaken(code, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrCouponCodeTaken
	}

	return repo.CreateFrom[models.Coupon](sql.H{
		"code":            code,
		"type":            in.Type,
		"value_cents":     in.ValueCents,
		"percent_off":     in.PercentOff,
		"threshold_cents": in.ThresholdCents,
		"total_count":     in.TotalCount,
		"starts_at":       in.StartsAt,
		"expires_at":      in.ExpiresAt,
		"enabled":         in.Enabled,
	})
}

func AdminUpdateCoupon(id sql.IdType, in CouponInput) error {
	code := in.normalizeCode(in.Type)
	if err := in.validate(code); err != nil {
		return err
	}
	if taken, err := couponCodeTaken(code, int64(id)); err != nil {
		return err
	} else if taken {
		return ErrCouponCodeTaken
	}

	return repo.UpdateByID[models.Coupon](id, sql.H{
		"code":            code,
		"type":            in.Type,
		"value_cents":     in.ValueCents,
		"percent_off":     in.PercentOff,
		"threshold_cents": in.ThresholdCents,
		"total_count":     in.TotalCount,
		"starts_at":       in.StartsAt,
		"expires_at":      in.ExpiresAt,
		"enabled":         in.Enabled,
		"updated_at":      time.Now().UTC(),
	})
}

func AdminSetCouponEnabled(id sql.IdType, enabled bool) error {
	return repo.UpdateByID[models.Coupon](id, sql.H{
		"enabled":    enabled,
		"updated_at": time.Now().UTC(),
	})
}

// ParseOptionalTime parses a datetime-local form value ("2006-01-02T15:04")
// into UTC, returning nil for blank input.
func ParseOptionalTime(input string) (*time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04", input, time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q", input)
	}
	utc := parsed.UTC()
	return &utc, nil
}
