package cart_api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/cart"
	"github.com/daqing/airway/lib/render"
)

func CartPageAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	lines, subtotal, err := services.CartLines(int64(user.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, cart.Page(cart.PageData{
		User:     user,
		Lines:    lines,
		Subtotal: services.FormatCents(subtotal),
		Flash:    c.Query("error"),
	}))
}

func AddAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	productID, err := strconv.ParseInt(c.PostForm("product_id"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid product")
		return
	}

	var variantID *int64
	if raw := c.PostForm("variant_id"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			variantID = &parsed
		}
	}

	quantity := 1
	if raw := c.PostForm("quantity"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			quantity = parsed
		}
	}

	back := c.PostForm("back")
	if back == "" || !strings.HasPrefix(back, "/") || strings.HasPrefix(back, "//") {
		back = "/cart"
	}

	if err := services.AddToCart(int64(user.ID), productID, variantID, quantity); err != nil {
		render.Found(c, back+"?error="+err.Error())
		return
	}

	render.Found(c, "/cart")
}

func UpdateItemAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, "/cart?error=invalid+item")
		return
	}

	quantity := 0
	if parsed, err := strconv.Atoi(c.PostForm("quantity")); err == nil {
		quantity = parsed
	}

	if err := services.UpdateCartItem(int64(user.ID), itemID, quantity); err != nil {
		render.Found(c, "/cart?error="+err.Error())
		return
	}

	render.Found(c, "/cart")
}

func RemoveItemAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, "/cart?error=invalid+item")
		return
	}

	if err := services.RemoveCartItem(int64(user.ID), itemID); err != nil {
		render.Found(c, "/cart?error="+err.Error())
		return
	}

	render.Found(c, "/cart")
}

func ClearAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	if err := services.ClearCart(int64(user.ID)); err != nil {
		render.Found(c, "/cart?error="+err.Error())
		return
	}

	render.Found(c, "/cart")
}
