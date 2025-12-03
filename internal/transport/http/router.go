package http

import (
	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	"github.com/gin-gonic/gin"
)

func NewRouter(handler *Handler, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(loggingMiddleware())

	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "ok")
	})

	router.Any("/oauth2/token-exchange/*path", handler.Check)

	return router
}
