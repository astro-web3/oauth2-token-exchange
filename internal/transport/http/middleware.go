package http

import (
	"net/http"
	"strings"
	"time"

	"log/slog"

	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/gin-gonic/gin"
)

const minServerErrorStatus = 500

const healthzPath = "/healthz"

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		if path == healthzPath {
			return
		}

		if status >= minServerErrorStatus {
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

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowedOrigins := make(map[string]bool)
	for _, origin := range cfg.CORS.AllowedOrigins {
		allowedOrigins[origin] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 只对 PAT 服务路径应用 CORS
		if !strings.HasPrefix(path, "/pat.v1.PATService") {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")

		// 检查 Origin 是否在允许列表中
		if origin != "" && allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if len(allowedOrigins) == 0 {
			// 如果没有配置允许的域名，允许所有域名（开发环境）
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// 设置允许的方法
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")

		// 设置允许的请求头，包含 Connect RPC 协议头
		c.Writer.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, "+
				"Authorization, accept, origin, Cache-Control, X-Requested-With, "+
				"Connect-Protocol-Version, Connect-Content-Encoding, Connect-Timeout-Ms, "+
				"X-Auth-Request-User, X-Auth-Request-Email, X-Auth-Request-Preferred-Username",
		)

		// 处理 OPTIONS 预检请求
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
