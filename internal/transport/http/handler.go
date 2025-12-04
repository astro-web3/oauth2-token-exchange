package http

import (
	"net/http"
	"strings"

	"log/slog"

	"github.com/astro-web3/oauth2-token-exchange/internal/app/authz"
	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

type Handler struct {
	appService authz.Service
	cfg        *config.Config
	headerKeys map[string]string
}

func NewHandler(appService authz.Service, cfg *config.Config) *Handler {
	return &Handler{
		appService: appService,
		cfg:        cfg,
		headerKeys: map[string]string{
			"user_id":                 cfg.Auth.HeaderKeys.UserID,
			"user_email":              cfg.Auth.HeaderKeys.UserEmail,
			"user_groups":             cfg.Auth.HeaderKeys.UserGroups,
			"user_preferred_username": cfg.Auth.HeaderKeys.UserPreferredUsername,
			"user_jwt":                cfg.Auth.HeaderKeys.UserJWT,
		},
	}
}

func (h *Handler) Check(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "transport.http.Check")
	defer span.End()

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		authHeader = c.GetHeader("authorization")
	}

	if authHeader == "" {
		span.SetAttributes(attribute.Bool("authz.missing_header", true))
		// c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
		c.Status(http.StatusOK)
		return
	}

	pat := strings.TrimPrefix(authHeader, "Bearer ")
	pat = strings.TrimSpace(pat)

	decision, err := h.appService.Check(ctx, pat, h.cfg.Auth.CacheTTL, h.headerKeys)

	if err != nil {
		span.RecordError(err)
		logger.ErrorContext(ctx, "failed to check authorization", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if !decision.Allow {
		span.SetAttributes(
			attribute.Bool("authz.allowed", false),
			attribute.String("authz.reason", decision.Reason),
		)
		logger.WarnContext(ctx, "authorization denied", slog.String("reason", decision.Reason))
		c.JSON(http.StatusUnauthorized, gin.H{"error": decision.Reason})
		return
	}

	span.SetAttributes(attribute.Bool("authz.allowed", true))

	for k, v := range decision.Headers {
		c.Header(k, v)
	}

	c.Status(http.StatusOK)
}
