package couponscenter_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/couponscenter"
	"github.com/daqing/airway/lib/render"
)

func CenterAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	claimable, err := services.ListClaimableCoupons(int64(user.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	mine, err := services.MyCoupons(int64(user.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, couponscenter.Center(couponscenter.CenterData{
		User:      user,
		Claimable: claimable,
		Mine:      mine,
		Flash:     c.Query("error"),
	}))
}

func ClaimAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	couponID, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil {
		render.Found(c, "/coupons?error=invalid+coupon")
		return
	}

	if err := services.ClaimCoupon(int64(user.ID), couponID); err != nil {
		render.Found(c, "/coupons?error="+err.Error())
		return
	}

	render.Found(c, "/coupons")
}
