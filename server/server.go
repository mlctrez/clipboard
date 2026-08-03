package server

import (
	"clipboard/api"
	"clipboard/static"
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kardianos/service"
)

const (
	authCookieName = "clip_auth"
	// ~10 years; browsers may clamp long-lived cookies.
	authCookieMaxAge = 10 * 365 * 24 * 60 * 60
)

type impl struct {
	db        api.StorageApi
	listener  net.Listener
	srv       *http.Server
	logger    service.Logger
	clipToken string
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

func (i *impl) isExternal(c *gin.Context) bool {
	return c.Request.Header.Get("X-Homessl-Forwarded") == "true"
}

func (i *impl) hasAdminCookie(c *gin.Context) bool {
	cookie, err := c.Cookie(authCookieName)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie), []byte(i.clipToken)) == 1
}

// isAdmin: full admin on the LAN without a cookie, or externally when the auth cookie is set.
func (i *impl) isAdmin(c *gin.Context) bool {
	return !i.isExternal(c) || i.hasAdminCookie(c)
}

func (i *impl) Listen(address string) (err error) {
	i.logger.Info("listen at", address)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		if i.isAdmin(c) {
			c.Next()
			return
		}

		// External, unauthenticated: only public image GETs and login.
		path := c.Request.URL.Path
		if c.Request.Method == http.MethodGet && strings.HasPrefix(path, "/login/") {
			c.Next()
			return
		}
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

	r.GET("/login/:token", i.login)
	static.SetupRoutes(r, i.isAdmin)
	r.GET("/clips", i.listClips)
	r.GET("/clips/:timestamp", i.getClip)
	r.DELETE("/clips/:timestamp", i.deleteClip)
	r.POST("/clips", i.saveClip)
	r.PUT("/clips/:timestamp", i.replaceClip)

	i.srv = &http.Server{Handler: r}

	i.listener, err = net.Listen("tcp4", address)

	return
}

func (i *impl) login(c *gin.Context) {
	token := c.Param("token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(i.clipToken)) == 1 {
		c.SetSameSite(http.SameSiteLaxMode)
		// path=/, secure, httpOnly — domain empty (current host)
		c.SetCookie(authCookieName, i.clipToken, authCookieMaxAge, "/", "", true, true)
	}
	// Match or not: always redirect to root; wrong token never sets the cookie.
	c.Redirect(http.StatusFound, "/")
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
	if err := i.db.Delete(timestamp); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
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
		return
	}
	c.Status(http.StatusCreated)
}

func (i *impl) replaceClip(c *gin.Context) {
	timestamp := c.Param("timestamp")
	if timestamp == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

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
	// Keep the original timestamp so it replaces the existing entry
	ci.TimeStamp = timestamp
	if err := i.db.Save(ci); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusOK)
}

type UploadClip struct {
	Clip string `json:"clip"`
}

func New(db api.StorageApi, logger service.Logger, clipToken string) api.ServerApi {
	return &impl{db: db, logger: logger, clipToken: clipToken}
}
