package admin_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/admin"
	"github.com/daqing/airway/lib/render"
)

// DashboardAction renders the admin landing page.
func DashboardAction(c *gin.Context) {
	stats, err := services.DashboardData()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, admin.Dashboard(middlewares.CurrentAdmin(c), stats))
}
