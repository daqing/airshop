package account_api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"
	sql "github.com/daqing/airway/lib/sql"

	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/models"
	"github.com/daqing/airshop/app/services"
	"github.com/daqing/airshop/app/views/account"
)

func listRedirect(err error) string {
	return "/account/addresses?error=" + err.Error()
}

func AddressesPageAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	list, err := services.ListAddresses(int64(user.ID))
	if err != nil {
		render.ErrorMessage(c, err.Error())
		return
	}

	render.HTML(c, account.Addresses(account.AddressesData{
		User:    user,
		Address: list,
		Flash:   c.Query("error"),
	}))
}

func NewAddressAction(c *gin.Context) {
	render.HTML(c, account.AddressForm(
		middlewares.CurrentUser(c), "New address", "/account/addresses", nil, "",
	))
}

func CreateAddressAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	in := addressInputFromForm(c)

	if _, err := services.CreateAddress(int64(user.ID), in); err != nil {
		renderAddressFormError(c, user, "New address", "/account/addresses", nil, in, err.Error())
		return
	}

	render.Found(c, "/account/addresses")
}

func EditAddressAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, listRedirect(services.ErrAddressNotFound))
		return
	}

	addr, err := services.FindAddress(int64(user.ID), id)
	if err != nil {
		render.Found(c, listRedirect(err))
		return
	}

	action := "/account/addresses/" + strconv.FormatInt(id, 10) + "/update"
	render.HTML(c, account.AddressForm(user, "Edit address", action, addr, ""))
}

func UpdateAddressAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, listRedirect(services.ErrAddressNotFound))
		return
	}

	in := addressInputFromForm(c)
	if err := services.UpdateAddress(int64(user.ID), id, in); err != nil {
		display := &models.Address{
			ID: sql.IdType(id), UserID: int64(user.ID),
			Recipient: in.Recipient, Phone: in.Phone,
			Province: in.Province, City: in.City, District: in.District,
			Street: in.Street, IsDefault: in.IsDefault,
		}
		action := "/account/addresses/" + strconv.FormatInt(id, 10) + "/update"
		renderAddressFormError(c, user, "Edit address", action, display, in, err.Error())
		return
	}

	render.Found(c, "/account/addresses")
}

func DeleteAddressAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, listRedirect(services.ErrAddressNotFound))
		return
	}

	if err := services.DeleteAddress(int64(user.ID), id); err != nil {
		render.Found(c, listRedirect(err))
		return
	}

	render.Found(c, "/account/addresses")
}

func SetDefaultAddressAction(c *gin.Context) {
	user := middlewares.CurrentUser(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.Found(c, listRedirect(services.ErrAddressNotFound))
		return
	}

	if err := services.SetDefaultAddress(int64(user.ID), id); err != nil {
		render.Found(c, listRedirect(err))
		return
	}

	render.Found(c, "/account/addresses")
}

func addressInputFromForm(c *gin.Context) services.AddressInput {
	return services.AddressInput{
		Recipient: c.PostForm("recipient"),
		Phone:     c.PostForm("phone"),
		Province:  c.PostForm("province"),
		City:      c.PostForm("city"),
		District:  c.PostForm("district"),
		Street:    c.PostForm("street"),
		IsDefault: c.PostForm("is_default") == "true",
	}
}

func renderAddressFormError(c *gin.Context, user *models.User, title, action string, a *models.Address, in services.AddressInput, msg string) {
	display := a
	if display == nil {
		display = &models.Address{
			UserID: int64(user.ID), Recipient: in.Recipient, Phone: in.Phone,
			Province: in.Province, City: in.City, District: in.District,
			Street: in.Street, IsDefault: in.IsDefault,
		}
	}

	render.HTMLStatus(c, 422, account.AddressForm(user, title, action, display, msg))
}
