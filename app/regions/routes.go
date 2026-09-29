package regions

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Routes registers the region JSON endpoints on an API router group:
//
//	GET /api/v1/regions                provinces
//	GET /api/v1/regions/:code/children children of a division
func Routes(r *gin.RouterGroup) {
	r.GET("/regions", ProvincesAction)
	r.GET("/regions/:code/children", ChildrenAction)
}

func ProvincesAction(c *gin.Context) {
	provinces, err := Provinces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, provinces)
}

func ChildrenAction(c *gin.Context) {
	children, err := Children(c.Param("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if children == nil {
		children = []Region{}
	}
	c.JSON(http.StatusOK, children)
}
