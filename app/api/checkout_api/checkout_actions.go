package checkout_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/checkout"
	"github.com/daqing/airway/lib/render"
)

func PageAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	lines, subtotal, err := services.CartLines(int64(user.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}
	if len(lines) == 0 {
		render.Found(c, "/cart")
		return
	}

	addresses, err := services.ListAddresses(int64(user.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, checkout.Page(checkout.PageData{
		User:      user,
		Lines:     lines,
		Subtotal:  services.FormatCents(subtotal),
		Addresses: addresses,
		Error:     c.Query("error"),
	}))
}

func PlaceOrderAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	addressID, err := strconv.ParseInt(c.PostForm("address_id"), 10, 64)
	if err != nil {
		render.Found(c, "/checkout?error=choose+a+shipping+address")
		return
	}

	order, err := services.PlaceOrder(int64(user.ID), addressID, c.PostForm("payment_method"))
	if err != nil {
		render.Found(c, "/checkout?error="+err.Error())
		return
	}

	render.Found(c, "/orders/"+order.OrderNo)
}
