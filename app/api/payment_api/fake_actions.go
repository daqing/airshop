package payment_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/errors"
	"github.com/daqing/airshop/app/views/payment"
	"github.com/daqing/airway/lib/render"
)

func fakeOrder(c *gin.Context) (*models.Order, *models.User, bool) {
	user := middlewares.CurrentUser(c)

	order, err := services.FindOrderByNo(int64(user.ID), c.Query("order"))
	if err != nil {
		render.HTMLStatus(c, 404, errors.NotFound(user))
		return nil, nil, false
	}
	return order, user, true
}

func FakeCashierAction(c *gin.Context) {
	order, user, ok := fakeOrder(c)
	if !ok {
		return
	}

	render.HTML(c, payment.FakeCashier(payment.FakeCashierData{
		User:  user,
		Order: order,
		Total: services.FormatCents(order.TotalCents),
		Flash: c.Query("error"),
	}))
}

func FakeConfirmAction(c *gin.Context) {
	order, _, ok := fakeOrder(c)
	if !ok {
		return
	}

	// The shared payment-success entry point: idempotent for duplicate
	// notifications and stock-deducting via the state machine (T5.4).
	if err := services.MarkOrderPaid(int64(order.ID)); err != nil {
		render.Found(c, "/pay/fake?order="+order.OrderNo+"&error="+err.Error())
		return
	}

	render.Found(c, "/orders/"+order.OrderNo)
}
