package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
)

const currentUserKey = "currentUser"

// LoadUser resolves the session cookie to a user (or nil) and stores it on
// the gin context for downstream handlers and views.
func LoadUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user *models.User

		if token, err := c.Cookie(services.SessionCookieName); err == nil && token != "" {
			user, err = services.SessionUser(token)
			if err != nil {
				user = nil
			}
		}

		c.Set(currentUserKey, user)
		c.Next()
	}
}

// CurrentUser returns the user resolved by LoadUser, or nil for guests.
func CurrentUser(c *gin.Context) *models.User {
	value, exists := c.Get(currentUserKey)
	if !exists || value == nil {
		return nil
	}

	user, _ := value.(*models.User)
	return user
}

// RequireUser redirects guests to the sign-in page, preserving the original
// location so sign-in can send them back.
func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) == nil {
			target := "/signin?next=" + c.Request.URL.RequestURI()
			c.Redirect(http.StatusFound, target)
			c.Abort()
			return
		}
		c.Next()
	}
}
