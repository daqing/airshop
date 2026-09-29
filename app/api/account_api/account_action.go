package account_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/views/account"
	"github.com/daqing/airway/lib/render"
)

func PageAction(c *gin.Context) {
	render.HTML(c, account.Page(middlewares.CurrentUser(c)))
}
