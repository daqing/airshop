package admin_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
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

	render.HTML(c, shipments.Page(middlewares.CurrentAdmin(c), shipments.PageData{
		PaidOrders: paid,
		Shipments:  list,
		Flash:      c.Query("error"),
	}))
}

func ShipOrderAction(c *gin.Context) {
	admin := middlewares.CurrentAdmin(c)
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, "/admin/shipments?error=invalid+order")
		return
	}

	carrier := c.PostForm("carrier")
	trackingNo := c.PostForm("tracking_no")
	if err := services.ShipOrder(orderID, carrier, trackingNo); err != nil {
		render.Found(c, "/admin/shipments?error="+err.Error())
		return
	}

	services.Audit(int64(admin.ID), "order.ship", "order", strconv.FormatInt(orderID, 10), carrier+" "+trackingNo)
	render.Found(c, "/admin/shipments")
}

func ShipmentInTransitAction(c *gin.Context) {
	shipmentStatusAction(c, services.ShipmentStatusInTransit)
}

func ShipmentDeliveredAction(c *gin.Context) {
	shipmentStatusAction(c, services.ShipmentStatusDelivered)
}

func AddShipmentEventAction(c *gin.Context) {
	shipmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, "/admin/shipments?error=invalid+shipment")
		return
	}

	if err := services.AddShipmentEvent(shipmentID, c.PostForm("description")); err != nil {
		render.Found(c, "/admin/shipments?error="+err.Error())
		return
	}

	render.Found(c, "/admin/shipments")
}

func shipmentStatusAction(c *gin.Context, status string) {
	admin := middlewares.CurrentAdmin(c)
	shipmentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, "/admin/shipments?error=invalid+shipment")
		return
	}

	if err := services.UpdateShipmentStatus(shipmentID, status); err != nil {
		render.Found(c, "/admin/shipments?error="+err.Error())
		return
	}

	services.Audit(int64(admin.ID), "shipment."+status, "shipment", strconv.FormatInt(shipmentID, 10), "")
	render.Found(c, "/admin/shipments")
}
