package config

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/assets"
	"github.com/daqing/airshop/app/middlewares"
	"github.com/daqing/airshop/app/views/errors"
	"github.com/daqing/airway/lib/jsbuild"
	"github.com/daqing/airway/lib/openapi"
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/utils"

	"github.com/daqing/airshop/app/api/account_api"
	"github.com/daqing/airshop/app/api/admin_api"
	"github.com/daqing/airshop/app/api/auth_api"
	"github.com/daqing/airshop/app/api/cart_api"
	"github.com/daqing/airshop/app/api/checkout_api"
	"github.com/daqing/airshop/app/api/couponscenter_api"
	"github.com/daqing/airshop/app/api/health_api"
	"github.com/daqing/airshop/app/api/home_api"
	"github.com/daqing/airshop/app/api/openapi_api"
	"github.com/daqing/airshop/app/api/orders_api"
	"github.com/daqing/airshop/app/api/payment_api"
	"github.com/daqing/airshop/app/api/products_api"
	"github.com/daqing/airshop/app/api/storage_api"
	regions_api "github.com/daqing/airshop/app/regions"
	"github.com/daqing/airway/app/websocket"
	"github.com/daqing/airway/lib/plugin"
)

// The openapi:generate command enumerates the routes of the binary it runs
// in; config owns that table, so it registers itself as the route source.
func init() {
	openapi.RegisterRouteSource(Routes)
}

// Routes registers every route — public and internal — at the root paths. This
// is the full router used when the app is served without a URL_PREFIX.
func Routes(r *gin.Engine) {
	r.Use(middlewares.LoadUser())

	PublicRoutes(r)
	AdminRoutes(r)
	HealthRoutes(r)
	FallbackRoutes(r)
}

// PublicRoutes registers the customer-facing routes: the home page, the
// WebSocket, and the API. When a URL_PREFIX is configured these answer only
// under the prefix; see App.Handler.
func PublicRoutes(r *gin.Engine) {
	r.GET("/", home_api.IndexAction)
	r.GET("/products", products_api.ListAction)
	r.GET("/products/:slug", products_api.ShowAction)
	r.GET("/signin", auth_api.SignInPageAction)
	r.POST("/signin/code", auth_api.SendCodeAction)
	r.POST("/signin", auth_api.SignInAction)
	r.POST("/signout", auth_api.SignOutAction)
	r.GET("/account", middlewares.RequireUser(), account_api.PageAction)
	r.GET("/cart", middlewares.RequireUser(), cart_api.CartPageAction)
	r.POST("/cart/add", middlewares.RequireUser(), cart_api.AddAction)
	r.POST("/cart/items/:id", middlewares.RequireUser(), cart_api.UpdateItemAction)
	r.POST("/cart/items/:id/delete", middlewares.RequireUser(), cart_api.RemoveItemAction)
	r.POST("/cart/clear", middlewares.RequireUser(), cart_api.ClearAction)
	r.GET("/checkout", middlewares.RequireUser(), checkout_api.PageAction)
	r.POST("/checkout", middlewares.RequireUser(), checkout_api.PlaceOrderAction)
	r.GET("/orders/:orderNo", middlewares.RequireUser(), orders_api.DetailAction)
	r.POST("/orders/:orderNo/cancel", middlewares.RequireUser(), orders_api.CancelAction)
	r.GET("/orders", middlewares.RequireUser(), orders_api.ListAction)
	r.GET("/coupons", middlewares.RequireUser(), couponscenter_api.CenterAction)
	r.POST("/coupons/claim", middlewares.RequireUser(), couponscenter_api.ClaimAction)
	r.GET("/pay/fake", middlewares.RequireUser(), payment_api.FakeCashierAction)
	r.POST("/pay/fake/confirm", middlewares.RequireUser(), payment_api.FakeConfirmAction)
	r.GET("/account/addresses", middlewares.RequireUser(), account_api.AddressesPageAction)
	r.GET("/account/addresses/new", middlewares.RequireUser(), account_api.NewAddressAction)
	r.POST("/account/addresses", middlewares.RequireUser(), account_api.CreateAddressAction)
	r.GET("/account/addresses/:id/edit", middlewares.RequireUser(), account_api.EditAddressAction)
	r.POST("/account/addresses/:id/update", middlewares.RequireUser(), account_api.UpdateAddressAction)
	r.POST("/account/addresses/:id/delete", middlewares.RequireUser(), account_api.DeleteAddressAction)
	r.POST("/account/addresses/:id/default", middlewares.RequireUser(), account_api.SetDefaultAddressAction)

	assetRoutes(r)
	websocketRoutes(r)
	apiGroupRoutes(r)
	openapiRoutes(r)

	plugin.MountAll(r)
}

// AdminRoutes registers the back-office routes under /admin. Admin pages
// render inside the admin layout; access control arrives with the admin
// auth work (T9.1).
func AdminRoutes(r *gin.Engine) {
	admin := r.Group("/admin")
	{
		admin.GET("", admin_api.DashboardAction)

		admin.GET("/categories", admin_api.CategoriesIndexAction)
		admin.GET("/categories/new", admin_api.NewCategoryAction)
		admin.POST("/categories", admin_api.CreateCategoryAction)
		admin.GET("/categories/:id/edit", admin_api.EditCategoryAction)
		admin.POST("/categories/:id/update", admin_api.UpdateCategoryAction)
		admin.POST("/categories/:id/delete", admin_api.DestroyCategoryAction)

		admin.GET("/products", admin_api.ProductsIndexAction)
		admin.GET("/products/new", admin_api.NewProductAction)
		admin.POST("/products", admin_api.CreateProductAction)
		admin.GET("/products/:id/edit", admin_api.EditProductAction)
		admin.POST("/products/:id/update", admin_api.UpdateProductAction)
		admin.POST("/products/:id/activate", admin_api.ActivateProductAction)
		admin.POST("/products/:id/deactivate", admin_api.DeactivateProductAction)
		admin.POST("/products/:id/images/upload", admin_api.UploadProductImagesAction)
		admin.POST("/products/:id/images/:imageId/delete", admin_api.DeleteProductImageAction)
		admin.POST("/products/:id/images/:imageId/main", admin_api.MakeProductImageMainAction)

		admin.GET("/coupons", admin_api.CouponsPageAction)
		admin.GET("/coupons/new", admin_api.NewCouponAction)
		admin.POST("/coupons", admin_api.CreateCouponAction)
		admin.GET("/coupons/:id/edit", admin_api.EditCouponAction)
		admin.POST("/coupons/:id/update", admin_api.UpdateCouponAction)
		admin.POST("/coupons/:id/enable", admin_api.EnableCouponAction)
		admin.POST("/coupons/:id/disable", admin_api.DisableCouponAction)
	}
}

// FallbackRoutes registers the no-route handler so unknown paths get the
// storefront 404 page instead of gin's plain-text default.
func FallbackRoutes(r *gin.Engine) {
	r.NoRoute(func(c *gin.Context) {
		render.HTMLStatus(c, 404, errors.NotFound(middlewares.CurrentUser(c)))
	})
}

// ForbiddenHandler renders the 403 page. Future auth middleware (admin
// accounts, T9.1) calls this to deny access.
func ForbiddenHandler(c *gin.Context) {
	render.HTMLStatus(c, 403, errors.Forbidden(middlewares.CurrentUser(c)))
}

// HealthRoutes registers the internal health-check route. It stays reachable at
// the unprefixed root (for load-balancer probes) even when the public routes
// are served under a URL_PREFIX.
func HealthRoutes(r *gin.Engine) {
	health_api.Routes(r)
}

func apiGroupRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		storage_api.Routes(v1)
		regions_api.Routes(v1)
	}
}

func websocketRoutes(r *gin.Engine) {
	r.GET("/ws", websocket.Conn)
	r.POST("/ws/publish", websocket.Publish)
}

// openapiRoutes serves the generated OpenAPI document at /openapi.json.
func openapiRoutes(r *gin.Engine) {
	openapi_api.Routes(r)
}

// assetRoutes serves the frontend bundle. In local development an in-memory
// esbuild server (started from main when AIRWAY_ENV=local) serves rebuilt
// output directly; otherwise the embedded production bundle answers.
func assetRoutes(r *gin.Engine) {
	if utils.AppConfig().IsLocal {
		if dev := jsbuild.Default(); dev != nil {
			r.GET("/assets/*path", gin.WrapH(dev.Handler()))
			return
		}
	}
	r.GET("/assets/*path", gin.WrapH(assets.Handler()))
}
