package services

import (
	"errors"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/daqing/airway/lib/repo"
	sql "github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"

	"github.com/daqing/airshop/app/models"
)

const (
	AdminSessionTTL     = 12 * time.Hour
	AdminSessionCookie  = "admin_session_token"
	AdminStatusActive   = "active"
	AdminStatusDisabled = "disabled"
)

var (
	ErrAdminLoginFailed   = errors.New("invalid username or password")
	ErrAdminNotFound      = errors.New("admin not found")
	ErrAdminDisabled      = errors.New("this admin account is disabled")
	ErrAdminUsernameTaken = errors.New("username is already taken")
)

// BootstrapAdminIfEmpty creates the first admin from ADMIN_USERNAME /
// ADMIN_PASSWORD (defaults admin / admin123 for local development) when the
// table has no rows.
func BootstrapAdminIfEmpty() error {
	count, err := repo.CountEvery[models.AdminUser]()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	username := strings.TrimSpace(utils.GetEnvMulti("ADMIN_USERNAME", ""))
	password := utils.GetEnvMulti("ADMIN_PASSWORD", "")
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin123"
		log.Println("no admin exists: bootstrapping default admin/admin123 — set ADMIN_USERNAME and ADMIN_PASSWORD before going live")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = repo.CreateFrom[models.AdminUser](sql.H{
		"username":      username,
		"password_hash": string(hash),
		"display_name":  "Administrator",
		"status":        AdminStatusActive,
	})
	return err
}

// AdminLogin verifies credentials and issues a session token.
func AdminLogin(username, password string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", ErrAdminLoginFailed
	}

	admin, err := repo.FindOneBy[models.AdminUser](sql.H{"username": username})
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", ErrAdminLoginFailed
	}
	if admin.Status != AdminStatusActive {
		return "", ErrAdminDisabled
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return "", ErrAdminLoginFailed
	}

	token := utils.RandomHex(32)
	if _, err := repo.CreateFrom[models.AdminSession](sql.H{
		"token":      token,
		"admin_id":   int64(admin.ID),
		"expires_at": time.Now().UTC().Add(AdminSessionTTL),
	}); err != nil {
		return "", err
	}
	return token, nil
}

// AdminSessionAdmin resolves an admin session token, mirroring SessionUser.
func AdminSessionAdmin(token string) (*models.AdminUser, error) {
	if token == "" {
		return nil, nil
	}

	session, err := repo.FindOneBy[models.AdminSession](sql.H{"token": token})
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	if time.Now().After(session.ExpiresAt) {
		_ = repo.DeleteByID[models.AdminSession](session.ID)
		return nil, nil
	}

	admin, err := repo.FindByID[models.AdminUser](sql.IdType(session.AdminID))
	if err != nil {
		return nil, err
	}
	if admin == nil || admin.Status != AdminStatusActive {
		return nil, nil
	}
	return admin, nil
}

// DestroyAdminSession removes the admin session row (sign-out).
func DestroyAdminSession(token string) error {
	if token == "" {
		return nil
	}
	return repo.DeleteWhere[models.AdminSession](sql.H{"token": token})
}
