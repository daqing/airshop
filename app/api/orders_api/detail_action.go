package orders_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/errors"
	"github.com/daqing/airshop/app/views/orders"
	"github.com/daqing/airway/lib/render"
)

func DetailAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	order, err := services.FindOrderByNo(int64(user.ID), c.Param("orderNo"))
	if err != nil {
		render.HTMLStatus(c, 404, errors.NotFound(user))
		return
	}

	items, err := services.OrderItems(int64(order.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, orders.Detail(orders.DetailData{
		User:  user,
		Order: order,
		Items: items,
	}))
}
