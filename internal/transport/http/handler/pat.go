package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	patapp "github.com/astro-web3/oauth2-token-exchange/internal/app/pat"
	patdomain "github.com/astro-web3/oauth2-token-exchange/internal/domain/pat"
	patv1 "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/pat/v1"
	patv1connect "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/pat/v1/patv1connect"
	"github.com/astro-web3/oauth2-token-exchange/pkg/logger"
	"github.com/astro-web3/oauth2-token-exchange/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
)

const (
	headerUserID              = "X-Auth-Request-User"
	headerEmail               = "X-Auth-Request-Email"
	headerPreferredUsername   = "X-Auth-Request-Preferred-Username"
)

type PATHandler struct {
	commandService *patapp.CommandService
	queryService   *patapp.QueryService
}

func NewPATHandler(
	commandService *patapp.CommandService,
	queryService *patapp.QueryService,
) patv1connect.PATServiceHandler {
	return &PATHandler{
		commandService: commandService,
		queryService:   queryService,
	}
}

func (h *PATHandler) CreatePAT(
	ctx context.Context,
	req *connect.Request[patv1.CreatePATRequest],
) (*connect.Response[patv1.CreatePATResponse], error) {
	ctx, span := tracer.Start(ctx, "transport.http.CreatePAT")
	defer span.End()

	userID := req.Header().Get(headerUserID)
	email := req.Header().Get(headerEmail)
	preferredUsername := req.Header().Get(headerPreferredUsername)

	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing X-Auth-Request-User header"))
	}

	expirationDate := time.Unix(req.Msg.GetExpirationDate(), 0)
	if expirationDate.Before(time.Now()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, patdomain.ErrInvalidExpiration)
	}

	span.SetAttributes(
		attribute.String("pat.user_id", userID),
		attribute.String("pat.email", email),
	)

	logger.InfoContext(ctx, "creating PAT",
		slog.String("user_id", userID),
		slog.String("email", email),
	)

	pat, token, err := h.commandService.CreatePAT(ctx, userID, email, preferredUsername, expirationDate)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, patdomain.ErrInvalidExpiration) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if errors.Is(err, patdomain.ErrFailedToCreatePAT) {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&patv1.CreatePATResponse{
		Pat: &patv1.PAT{
			Id:             pat.ID,
			UserId:         pat.UserID,
			ExpirationDate: pat.ExpirationDate.Unix(),
			CreatedAt:      pat.CreatedAt.Unix(),
		},
		Token: token,
	}), nil
}

func (h *PATHandler) ListPATs(
	ctx context.Context,
	req *connect.Request[patv1.ListPATsRequest],
) (*connect.Response[patv1.ListPATsResponse], error) {
	ctx, span := tracer.Start(ctx, "transport.http.ListPATs")
	defer span.End()

	userID := req.Header().Get(headerUserID)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing X-Auth-Request-User header"))
	}

	span.SetAttributes(attribute.String("pat.user_id", userID))

	logger.InfoContext(ctx, "listing PATs", slog.String("user_id", userID))

	pats, err := h.queryService.ListPATs(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	patProtos := make([]*patv1.PAT, 0, len(pats))
	for _, pat := range pats {
		patProtos = append(patProtos, &patv1.PAT{
			Id:             pat.ID,
			UserId:         pat.UserID,
			ExpirationDate: pat.ExpirationDate.Unix(),
			CreatedAt:      pat.CreatedAt.Unix(),
		})
	}

	return connect.NewResponse(&patv1.ListPATsResponse{
		Pats: patProtos,
	}), nil
}

func (h *PATHandler) DeletePAT(
	ctx context.Context,
	req *connect.Request[patv1.DeletePATRequest],
) (*connect.Response[patv1.DeletePATResponse], error) {
	ctx, span := tracer.Start(ctx, "transport.http.DeletePAT")
	defer span.End()

	userID := req.Header().Get(headerUserID)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing X-Auth-Request-User header"))
	}

	patID := req.Msg.GetPatId()

	span.SetAttributes(
		attribute.String("pat.user_id", userID),
		attribute.String("pat.id", patID),
	)

	logger.InfoContext(ctx, "deleting PAT",
		slog.String("user_id", userID),
		slog.String("pat_id", patID),
	)

	err := h.commandService.DeletePAT(ctx, userID, patID)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, patdomain.ErrPATNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, patdomain.ErrMachineUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&patv1.DeletePATResponse{
		Success: true,
	}), nil
}

