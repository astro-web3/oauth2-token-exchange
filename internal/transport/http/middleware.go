package http

import (
	"time"

	"log/slog"

	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/gin-gonic/gin"
)

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		if status >= 500 {
			logger.ErrorContext(c.Request.Context(), "request failed",
				slog.String("method", method),
				slog.String("path", path),
				slog.Int("status", status),
				slog.Duration("duration", duration),
			)
		} else {
			logger.InfoContext(c.Request.Context(), "request completed",
				slog.String("method", method),
				slog.String("path", path),
				slog.Int("status", status),
				slog.Duration("duration", duration),
			)
		}
	}
}
