package http

import (
	"net/http"

	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	patv1connect "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/pat/v1/patv1connect"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func NewRouter(handler *Handler, cfg *config.Config, patHandler patv1connect.PATServiceHandler) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	router.Use(gin.Recovery())
	if cfg.Observability.TraceEnabled {
		router.Use(otelgin.Middleware(serviceName))
	}
	router.Use(loggingMiddleware())

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	router.Any("/oauth2/token-exchange/*path", handler.Check)

	patServicePath, patServiceHandler := patv1connect.NewPATServiceHandler(patHandler)
	router.Any(patServicePath+"/*method", gin.WrapH(patServiceHandler))

	return router
}
