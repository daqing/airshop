package auth_api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/auth"
	"github.com/daqing/airway/lib/render"
)

func SignInPageAction(c *gin.Context) {
	render.HTML(c, auth.SignIn(auth.SignInData{
		Phone: c.Query("phone"),
		Sent:  c.Query("sent") == "1",
		Error: c.Query("error"),
	}))
}

func SendCodeAction(c *gin.Context) {
	phone, err := services.NormalizePhone(c.PostForm("phone"))
	if err != nil {
		redirectToSignIn(c, c.PostForm("phone"), err.Error())
		return
	}

	if _, err := services.SendSignInCode(phone); err != nil {
		redirectToSignIn(c, phone, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/signin?phone="+phone+"&sent=1")
}

func SignInAction(c *gin.Context) {
	phone, err := services.NormalizePhone(c.PostForm("phone"))
	if err != nil {
		redirectToSignIn(c, c.PostForm("phone"), err.Error())
		return
	}

	if err := services.ConsumeSignInCode(phone, c.PostForm("code")); err != nil {
		redirectToSignIn(c, phone, err.Error())
		return
	}

	user, err := services.SignInOrSignUp(phone)
	if err != nil {
		redirectToSignIn(c, phone, err.Error())
		return
	}

	token, err := services.CreateSession(int64(user.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	c.SetCookie(services.SessionCookieName, token, int(services.SessionTTL.Seconds()), "/", "", false, true)
	render.Found(c, "/")
}

func SignOutAction(c *gin.Context) {
	if token, err := c.Cookie(services.SessionCookieName); err == nil && token != "" {
		_ = services.DestroySession(token)
	}

	c.SetCookie(services.SessionCookieName, "", -1, "/", "", false, true)
	render.Found(c, "/")
}

func redirectToSignIn(c *gin.Context, phone, message string) {
	c.Redirect(http.StatusFound, "/signin?phone="+phone+"&error="+message)
}
