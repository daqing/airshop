package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
)

const currentAdminKey = "currentAdmin"

// LoadAdminUser resolves the admin session cookie on every request so admin
// handlers and views can access the signed-in admin.
func LoadAdminUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var admin *models.AdminUser

		if token, err := c.Cookie(services.AdminSessionCookie); err == nil && token != "" {
			admin, err = services.AdminSessionAdmin(token)
			if err != nil {
				admin = nil
			}
		}

		c.Set(currentAdminKey, admin)
		c.Next()
	}
}

// CurrentAdmin returns the admin resolved by LoadAdminUser, or nil.
func CurrentAdmin(c *gin.Context) *models.AdminUser {
	value, exists := c.Get(currentAdminKey)
	if !exists || value == nil {
		return nil
	}

	admin, _ := value.(*models.AdminUser)
	return admin
}

// RequireAdmin bounces guests and customers to the admin sign-in page.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentAdmin(c) == nil {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}
		c.Next()
	}
}
