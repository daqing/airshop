package services

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"

	"github.com/daqing/airshop/app/models"
)

const (
	signInCodeTTL     = 5 * time.Minute
	SessionTTL        = 30 * 24 * time.Hour
	SessionCookieName = "session_token"
)

var phoneNumberPattern = regexp.MustCompile(`^\d{7,15}$`)

var (
	ErrPhoneInvalid    = errors.New("enter a valid phone number")
	ErrCodeRequired    = errors.New("enter the verification code")
	ErrCodeInvalid     = errors.New("verification code is invalid or expired")
	ErrAccountDisabled = errors.New("this account is disabled")
)

type codeEntry struct {
	code      string
	expiresAt time.Time
	tries     int
}

var (
	codeStoreMu sync.Mutex
	codeStore   = map[string]codeEntry{}
)

// NormalizePhone strips common separators and validates the digits.
func NormalizePhone(input string) (string, error) {
	phone := strings.ReplaceAll(strings.TrimSpace(input), " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.TrimPrefix(phone, "+86")
	if !phoneNumberPattern.MatchString(phone) {
		return "", ErrPhoneInvalid
	}
	return phone, nil
}

// SendSignInCode generates a verification code for the phone number. Per the
// T2.2 decision there is no SMS provider yet: the code is logged to the
// server console and kept in an in-memory store.
func SendSignInCode(input string) (string, error) {
	phone, err := NormalizePhone(input)
	if err != nil {
		return "", err
	}

	pruneExpiredCodes()

	n, err := crand.Int(crand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", n.Int64())

	codeStoreMu.Lock()
	codeStore[phone] = codeEntry{code: code, expiresAt: time.Now().Add(signInCodeTTL)}
	codeStoreMu.Unlock()

	log.Printf("sign-in code for %s: %s", phone, code)
	return code, nil
}

// pruneExpiredCodes drops stale entries so the in-memory store cannot grow
// without bound.
func pruneExpiredCodes() {
	now := time.Now()

	codeStoreMu.Lock()
	defer codeStoreMu.Unlock()
	for phone, entry := range codeStore {
		if now.After(entry.expiresAt) {
			delete(codeStore, phone)
		}
	}
}

func takeCodeEntry(phone string) (codeEntry, bool) {
	codeStoreMu.Lock()
	defer codeStoreMu.Unlock()

	entry, ok := codeStore[phone]
	if ok {
		delete(codeStore, phone)
	}
	return entry, ok
}

// ConsumeSignInCode verifies and removes the code for the phone number.
func ConsumeSignInCode(input, code string) error {
	phone, err := NormalizePhone(input)
	if err != nil {
		return err
	}

	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return ErrCodeRequired
	}

	entry, ok := takeCodeEntry(phone)
	if !ok || time.Now().After(entry.expiresAt) || entry.code != trimmed {
		return ErrCodeInvalid
	}
	return nil
}

// SignInOrSignUp returns the active user for the phone number, creating the
// account on first sign-in.
func SignInOrSignUp(input string) (*models.User, error) {
	phone, err := NormalizePhone(input)
	if err != nil {
		return nil, err
	}

	user, err := repo.FindOneBy[models.User](sql.H{"phone_number": phone})
	if err != nil {
		return nil, err
	}

	if user == nil {
		user, err = repo.CreateFrom[models.User](sql.H{
			"phone_number": phone,
			"display_name": "User " + phone[len(phone)-4:],
			"status":       "active",
		})
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	if user.Status != "active" {
		return nil, ErrAccountDisabled
	}
	return user, nil
}

// CreateSession issues a new session token for the user.
func CreateSession(userID int64) (string, error) {
	token := utils.RandomHex(32)
	if _, err := repo.CreateFrom[models.Session](sql.H{
		"token":      token,
		"user_id":    userID,
		"expires_at": time.Now().UTC().Add(SessionTTL),
	}); err != nil {
		return "", err
	}
	return token, nil
}

// SessionUser resolves a session token to its active user. Expired or
// unknown tokens yield nil; expired rows are cleaned up as they are seen.
func SessionUser(token string) (*models.User, error) {
	if token == "" {
		return nil, nil
	}

	session, err := repo.FindOneBy[models.Session](sql.H{"token": token})
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	if time.Now().After(session.ExpiresAt) {
		_ = repo.DeleteByID[models.Session](session.ID)
		return nil, nil
	}

	user, err := repo.FindByID[models.User](sql.IdType(session.UserID))
	if err != nil {
		return nil, err
	}
	if user == nil || user.Status != "active" {
		return nil, nil
	}
	return user, nil
}

// DestroySession removes the session row (sign-out).
func DestroySession(token string) error {
	if token == "" {
		return nil
	}
	return repo.DeleteWhere[models.Session](sql.H{"token": token})
}
