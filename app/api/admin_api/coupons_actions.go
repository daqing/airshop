package admin_api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/admin/coupons"
)

func CouponsPageAction(c *gin.Context) {
	list, err := services.AdminListCoupons()
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	usage := make(map[int64]int64, len(list))
	for _, cp := range list {
		count, err := services.CouponUsage(int64(cp.ID))
		if err != nil {
			render.ErrorMessage(c, err.Error())
			return
		}
		usage[int64(cp.ID)] = count
	}

	render.HTML(c, coupons.List(middlewares.CurrentAdmin(c), coupons.ListData{
		Coupons: list,
		Usage:   usage,
		Flash:   c.Query("error"),
	}))
}

func NewCouponAction(c *gin.Context) {
	render.HTML(c, coupons.CouponForm(middlewares.CurrentAdmin(c), "New coupon", "/admin/coupons", nil, ""))
}

func CreateCouponAction(c *gin.Context) {
	in := couponInputFromForm(c)

	created, err := services.AdminCreateCoupon(in)
	if err != nil {
		renderCouponFormError(c, "New coupon", "/admin/coupons", nil, in, err.Error())
		return
	}

	render.Found(c, "/admin/coupons/"+strconv.FormatInt(int64(created.ID), 10)+"/edit")
}

func EditCouponAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	cp, err := services.FindCoupon(id)
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}
	if cp == nil {
		render.Found(c, "/admin/coupons?error="+services.ErrCouponNotFound.Error())
		return
	}

	action := "/admin/coupons/" + strconv.FormatInt(int64(id), 10) + "/update"
	render.HTML(c, coupons.CouponForm(middlewares.CurrentAdmin(c), "Edit coupon", action, cp, ""))
}

func UpdateCouponAction(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	in := couponInputFromForm(c)
	if err := services.AdminUpdateCoupon(id, in); err != nil {
		display := &models.Coupon{
			ID: id, Code: strings.ToUpper(strings.TrimSpace(in.Code)), Type: in.Type,
			ValueCents: in.ValueCents, PercentOff: in.PercentOff, ThresholdCents: in.ThresholdCents,
			TotalCount: in.TotalCount, StartsAt: in.StartsAt, ExpiresAt: in.ExpiresAt, Enabled: in.Enabled,
		}
		action := "/admin/coupons/" + strconv.FormatInt(int64(id), 10) + "/update"
		renderCouponFormError(c, "Edit coupon", action, display, in, err.Error())
		return
	}

	render.Found(c, "/admin/coupons")
}

func EnableCouponAction(c *gin.Context) {
	toggleCouponEnabled(c, true)
}

func DisableCouponAction(c *gin.Context) {
	toggleCouponEnabled(c, false)
}

func toggleCouponEnabled(c *gin.Context, enabled bool) {
	id, err := parseID(c)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	if err := services.AdminSetCouponEnabled(id, enabled); err != nil {
		render.Found(c, "/admin/coupons?error="+err.Error())
		return
	}

	render.Found(c, "/admin/coupons")
}

func couponInputFromForm(c *gin.Context) services.CouponInput {
	in := services.CouponInput{
		Code:    c.PostForm("code"),
		Type:    c.PostForm("type"),
		Enabled: c.PostForm("enabled") == "true",
	}

	if cents, err := services.ParsePriceToCents(c.PostForm("value")); err == nil {
		in.ValueCents = cents
	}
	if percent, err := strconv.Atoi(c.PostForm("percent_off")); err == nil {
		in.PercentOff = percent
	}
	if cents, err := services.ParsePriceToCents(c.PostForm("threshold")); err == nil {
		in.ThresholdCents = cents
	}
	if count, err := strconv.Atoi(c.PostForm("total_count")); err == nil {
		in.TotalCount = count
	}
	if starts, err := services.ParseOptionalTime(c.PostForm("starts_at")); err == nil {
		in.StartsAt = starts
	}
	if expires, err := services.ParseOptionalTime(c.PostForm("expires_at")); err == nil {
		in.ExpiresAt = expires
	}
	return in
}

func renderCouponFormError(c *gin.Context, title, action string, cp *models.Coupon, in services.CouponInput, msg string) {
	display := cp
	if display == nil {
		display = &models.Coupon{
			Code: strings.ToUpper(strings.TrimSpace(in.Code)), Type: in.Type,
			ValueCents: in.ValueCents, PercentOff: in.PercentOff, ThresholdCents: in.ThresholdCents,
			TotalCount: in.TotalCount, StartsAt: in.StartsAt, ExpiresAt: in.ExpiresAt, Enabled: in.Enabled,
		}
	}

	render.HTMLStatus(c, 422, coupons.CouponForm(middlewares.CurrentAdmin(c), title, action, display, msg))
}
