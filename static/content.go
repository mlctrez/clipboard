package static

import (
	"embed"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed *
var staticContent embed.FS

func SetupRoutes(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		if c.Request.Header.Get("X-Homessl-Forwarded") == "true" {
			c.Params = []gin.Param{{Key: "file", Value: "external.html"}}
		} else {
			c.Params = []gin.Param{{Key: "file", Value: "index.html"}}
		}
		ServeContent(c)
	})
	r.GET("/:file", ServeContent)
}

func ServeContent(c *gin.Context) {
	path := filepath.Clean(c.Param("file"))
	if file, err := staticContent.ReadFile(fmt.Sprintf("%s", path)); os.IsNotExist(err) {
		c.AbortWithStatus(404)
	} else {
		if path == "index.html" {
			externalHost := os.Getenv("EXTERNAL_HOST")
			file = []byte(strings.ReplaceAll(string(file), "EXTERNAL_HOST", externalHost))
		}
		c.Data(200, mime.TypeByExtension(filepath.Ext(path)), file)
	}
}
