package admin_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/admin/shipments"
	"github.com/daqing/airway/lib/render"
)

func ShipmentsPageAction(c *gin.Context) {
	paid, err := services.ListPaidOrders()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	list, err := services.ListShipments()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, shipments.Page(shipments.PageData{
		PaidOrders: paid,
		Shipments:  list,
		Flash:      c.Query("error"),
	}))
}

func ShipOrderAction(c *gin.Context) {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, "/admin/shipments?error=invalid+order")
		return
	}

	if err := services.ShipOrder(orderID, c.PostForm("carrier"), c.PostForm("tracking_no")); err != nil {
		render.Found(c, "/admin/shipments?error="+err.Error())
		return
	}

	render.Found(c, "/admin/shipments")
}

func ShipmentInTransitAction(c *gin.Context) {
	shipmentStatusAction(c, services.ShipmentStatusInTransit)
}

func ShipmentDeliveredAction(c *gin.Context) {
	shipmentStatusAction(c, services.ShipmentStatusDelivered)
}

func shipmentStatusAction(c *gin.Context, status string) {
	shipmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, "/admin/shipments?error=invalid+shipment")
		return
	}

	if err := services.UpdateShipmentStatus(shipmentID, status); err != nil {
		render.Found(c, "/admin/shipments?error="+err.Error())
		return
	}

	render.Found(c, "/admin/shipments")
}
