package home_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/home"
	"github.com/daqing/airway/lib/render"
)

const homeProductsLimit = 8

func IndexAction(c *gin.Context) {
	products, err := services.LatestProducts(homeProductsLimit)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, home.Index(middlewares.CurrentUser(c), products))
}
