package orders_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	ordersview "github.com/daqing/airshop/app/views/orders"
	"github.com/daqing/airway/lib/render"
)

func ListAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	status := c.Query("status")
	page := 1
	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}

	orders, total, err := services.ListOrders(int64(user.ID), status, page, services.OrdersPageSize)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	totalPages := int((total + services.OrdersPageSize - 1) / services.OrdersPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	render.HTML(c, ordersview.List(ordersview.ListData{
		User:       user,
		Orders:     orders,
		Status:     status,
		Page:       page,
		TotalPages: totalPages,
		Total:      total,
	}))
}
