package orders_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/errors"
	ordersview "github.com/daqing/airshop/app/views/orders"
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

	render.HTML(c, ordersview.Detail(ordersview.DetailData{
		User:  user,
		Order: order,
		Items: items,
		Flash: c.Query("error"),
		Paid:  c.Query("paid") == "1",
	}))
}

func CancelAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	orderNo := c.Param("orderNo")

	if err := services.CancelOrder(int64(user.ID), orderNo); err != nil {
		render.Found(c, "/orders/"+orderNo+"?error="+err.Error())
		return
	}

	render.Found(c, "/orders/"+orderNo)
}
