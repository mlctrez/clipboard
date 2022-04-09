package server

import (
	"clipboard/api"
	"clipboard/static"
	"context"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kardianos/service"
)

type impl struct {
	db       api.StorageApi
	listener net.Listener
	srv      *http.Server
	logger   service.Logger
}

func (i *impl) Serve() error {
	i.logger.Info("serve")
	err := i.srv.Serve(i.listener)
	if err != http.ErrServerClosed {
		i.logger.Warning("Serve exited with error", err)
	}
	return err
}

func (i *impl) Shutdown(ctx context.Context) error {
	i.logger.Info("shutdown")
	return i.srv.Shutdown(ctx)
}

func (i *impl) Listen(address string) (err error) {
	i.logger.Info("listen at", address)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {

		// allow everything inside the network
		if c.Request.Header.Get("X-Homessl-Forwarded") != "true" {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		// only allow get /clips/timestamp outside network
		if c.Request.Method == http.MethodGet && strings.HasPrefix(path, "/clips/") {
			c.Next()
			return
		}
		if strings.HasPrefix(path, "/clips") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	})
	static.SetupRoutes(r)
	r.GET("/clips", i.listClips)
	r.GET("/clips/:timestamp", i.getClip)
	r.DELETE("/clips/:timestamp", i.deleteClip)
	r.POST("/clips", i.saveClip)

	i.srv = &http.Server{Handler: r}

	i.listener, err = net.Listen("tcp4", address)

	return
}

func (i *impl) listClips(c *gin.Context) {
	if list, err := i.db.List(); err != nil {
		_ = c.AbortWithError(500, err)
	} else {
		// display most recent at top
		sort.Sort(sort.Reverse(sort.StringSlice(list)))
		c.JSON(200, map[string][]string{"clips": list})
	}
}

func (i *impl) getClip(c *gin.Context) {
	timestamp := c.Param("timestamp")
	if timestamp == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	ci, err := i.db.Get(timestamp)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if ci == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Data(http.StatusOK, ci.ContentType, ci.Data)
}

func (i *impl) deleteClip(c *gin.Context) {
	timestamp := c.Param("timestamp")
	if timestamp == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	err := i.db.Delete(timestamp)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (i *impl) saveClip(c *gin.Context) {
	upload := &UploadClip{}
	if c.BindJSON(upload) != nil {
		return
	}

	var ci *api.ClippedImage
	var err error

	if ci, err = api.ParseClippedImage(upload.Clip); err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if ci == nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if err := i.db.Save(ci); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}

}

type UploadClip struct {
	Clip string `json:"clip"`
}

func New(db api.StorageApi, logger service.Logger) api.ServerApi {
	return &impl{db: db, logger: logger}
}
