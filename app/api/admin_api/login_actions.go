package admin_api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/admin"
	"github.com/daqing/airway/lib/render"
)

// LoginPageAction renders the admin sign-in page. Signed-in admins are sent
// straight to the dashboard.
func LoginPageAction(c *gin.Context) {
	if middlewares.CurrentAdmin(c) != nil {
		render.Found(c, "/admin")
		return
	}

	render.HTML(c, admin.Login(admin.LoginData{
		Username: c.Query("username"),
		Error:    c.Query("error"),
	}))
}

// LoginAction verifies credentials and issues the admin session cookie.
func LoginAction(c *gin.Context) {
	token, err := services.AdminLogin(c.PostForm("username"), c.PostForm("password"))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/login?username="+c.PostForm("username")+"&error="+err.Error())
		return
	}

	c.SetCookie(services.AdminSessionCookie, token, int(services.AdminSessionTTL.Seconds()), "/admin", "", false, true)
	render.Found(c, "/admin")
}

// LogoutAction destroys the admin session and clears the cookie.
func LogoutAction(c *gin.Context) {
	if token, err := c.Cookie(services.AdminSessionCookie); err == nil && token != "" {
		_ = services.DestroyAdminSession(token)
	}

	c.SetCookie(services.AdminSessionCookie, "", -1, "/admin", "", false, true)
	render.Found(c, "/admin/login")
}
