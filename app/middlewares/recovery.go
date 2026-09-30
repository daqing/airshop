package middlewares

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airshop/app/views/errors"
	"github.com/daqing/airway/lib/render"
)

// Recovery converts panics into the storefront 500 page with the real cause
// on the server log, instead of gin's plain-text default.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic on %s %s: %v", c.Request.Method, c.Request.URL.Path, rec)
				c.Abort()
				renderServerErrorPage(c)
			}
		}()
		c.Next()
	}
}

// renderServerErrorPage renders the 500 page; failures while rendering it
// fall back to a plain body so the recovery handler cannot panic again.
func renderServerErrorPage(c *gin.Context) {
	if c.Writer.Written() {
		return
	}
	c.Status(http.StatusInternalServerError)

	defer func() {
		if recover() != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error")
		}
	}()

	render.HTMLStatus(c, http.StatusInternalServerError, errors.ServerError(CurrentUser(c)))
}
