package admin_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/admin/orders"
	"github.com/daqing/airway/lib/render"
)

func OrdersPageAction(c *gin.Context) {
	status := c.Query("status")
	search := c.Query("q")
	page := 1
	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}

	list, total, err := services.AdminListOrders(status, search, page, services.AdminOrdersPageSize)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	totalPages := int((total + services.AdminOrdersPageSize - 1) / services.AdminOrdersPageSize)
	if totalPages < 1 {
		totalPages = 1
	}

	render.HTML(c, orders.List(orders.ListData{
		Admin:      middlewares.CurrentAdmin(c),
		Orders:     list,
		Status:     status,
		Search:     search,
		Page:       page,
		TotalPages: totalPages,
		Total:      total,
	}))
}

func OrderDetailAction(c *gin.Context) {
	order, err := services.AdminFindOrderByNo(c.Param("orderNo"))
	if err != nil {
		render.Found(c, "/admin/orders?error="+err.Error())
		return
	}

	items, err := services.OrderItems(int64(order.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	tracking := ""
	shipment, err := services.FindShipmentByOrder(int64(order.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}
	if shipment != nil {
		tracking = shipment.Carrier + " " + shipment.TrackingNo + " (" + shipment.Status + ")"
	}

	render.HTML(c, orders.Detail(orders.DetailData{
		Admin:    middlewares.CurrentAdmin(c),
		Order:    order,
		Items:    items,
		Flash:    c.Query("error"),
		Tracking: tracking,
	}))
}

func RefundOrderAction(c *gin.Context) {
	admin := middlewares.CurrentAdmin(c)
	orderNo := c.Param("orderNo")

	if err := services.AdminRefundOrder(orderNo); err != nil {
		render.Found(c, "/admin/orders/"+orderNo+"?error="+err.Error())
		return
	}

	services.Audit(int64(admin.ID), "order.refund", "order", orderNo, "")
	render.Found(c, "/admin/orders/"+orderNo)
}
