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

// SetupRoutes registers static file routes. isAdmin selects the full admin UI
// (index.html) vs the empty external page (external.html) for "/".
func SetupRoutes(r *gin.Engine, isAdmin func(*gin.Context) bool) {
	r.GET("/", func(c *gin.Context) {
		if isAdmin(c) {
			c.Params = []gin.Param{{Key: "file", Value: "index.html"}}
		} else {
			c.Params = []gin.Param{{Key: "file", Value: "external.html"}}
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
			// Required at service startup; host only (e.g. clipboard.mlctrez.com).
			externalHost := strings.TrimSpace(os.Getenv("EXTERNAL_HOST"))
			file = []byte(strings.ReplaceAll(string(file), "EXTERNAL_HOST", externalHost))
		}
		c.Data(200, mime.TypeByExtension(filepath.Ext(path)), file)
	}
}
