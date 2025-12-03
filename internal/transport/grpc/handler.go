package grpc

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"connectrpc.com/connect"
	"github.com/astro-web3/oauth2-token-exchange/internal/app/authz"
	"github.com/astro-web3/oauth2-token-exchange/internal/config"
	authv3 "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/envoy/service/auth/v3"
	authv3connect "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/envoy/service/auth/v3/authv3connect"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
)

type Handler struct {
	appService authz.Service
	cfg        *config.Config
	headerKeys map[string]string
}

func NewHandler(appService authz.Service, cfg *config.Config) authv3connect.AuthorizationHandler {
	return &Handler{
		appService: appService,
		cfg:        cfg,
		headerKeys: map[string]string{
			"user_id":     cfg.Auth.HeaderKeys.UserID,
			"user_email":  cfg.Auth.HeaderKeys.UserEmail,
			"user_groups": cfg.Auth.HeaderKeys.UserGroups,
			"user_jwt":    cfg.Auth.HeaderKeys.UserJWT,
		},
	}
}

func (h *Handler) Check(ctx context.Context, req *connect.Request[authv3.CheckRequest]) (*connect.Response[authv3.CheckResponse], error) {
	ctx, span := tracer.Start(ctx, "transport.grpc.Check")
	defer span.End()

	httpReq := req.Msg.GetAttributes().GetRequest().GetHttp()
	if httpReq == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing HTTP request"))
	}

	headers := httpReq.GetHeaders()
	authHeader := headers["authorization"]
	if authHeader == "" {
		authHeader = headers["Authorization"]
	}

	if authHeader == "" {
		span.SetAttributes(attribute.Bool("authz.missing_header", true))
		logger.WarnContext(ctx, "missing authorization header")
		return h.deniedResponse(401, "missing authorization header"), nil
	}

	pat := strings.TrimPrefix(authHeader, "Bearer ")
	pat = strings.TrimSpace(pat)

	decision, err := h.appService.Check(ctx, pat, h.cfg.Auth.CacheTTL, h.headerKeys)
	if err != nil {
		span.RecordError(err)
		logger.ErrorContext(ctx, "failed to check authorization", slog.String("error", err.Error()))
		return h.deniedResponse(500, "internal server error"), nil
	}

	if !decision.Allow {
		span.SetAttributes(
			attribute.Bool("authz.allowed", false),
			attribute.String("authz.reason", decision.Reason),
		)
		logger.WarnContext(ctx, "authorization denied", slog.String("reason", decision.Reason))
		return h.deniedResponse(401, decision.Reason), nil
	}

	span.SetAttributes(attribute.Bool("authz.allowed", true))
	logger.InfoContext(ctx, "authorization allowed")

	return h.allowedResponse(decision.Headers), nil
}

func (h *Handler) allowedResponse(headers map[string]string) *connect.Response[authv3.CheckResponse] {
	headerValueOptions := make([]*authv3.HeaderValueOption, 0, len(headers))
	for k, v := range headers {
		headerValueOptions = append(headerValueOptions, &authv3.HeaderValueOption{
			Header: &authv3.HeaderValue{
				Key:   k,
				Value: v,
			},
			Append: false,
		})
	}

	return connect.NewResponse(&authv3.CheckResponse{
		HttpResponse: &authv3.HttpResponse{
			Status: &authv3.HttpResponse_OkResponse{
				OkResponse: &authv3.OkHttpResponse{
					Headers: headerValueOptions,
				},
			},
		},
	})
}

func (h *Handler) deniedResponse(statusCode int32, reason string) *connect.Response[authv3.CheckResponse] {
	return connect.NewResponse(&authv3.CheckResponse{
		HttpResponse: &authv3.HttpResponse{
			Status: &authv3.HttpResponse_DeniedResponse{
				DeniedResponse: &authv3.DeniedHttpResponse{
					Status: statusCode,
					Body:   reason,
				},
			},
		},
	})
}
