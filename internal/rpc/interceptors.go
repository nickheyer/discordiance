package rpc

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

type loggingInterceptor struct{}

func (i *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		slog.Info("rpc request", "peer", req.Peer().Addr, "procedure", req.Spec().Procedure)

		resp, err := next(ctx, req)
		elapsed := time.Since(start)

		if err != nil {
			slog.Error("rpc error", "procedure", req.Spec().Procedure, "error", err, "duration", elapsed)
		} else {
			slog.Info("rpc response", "procedure", req.Spec().Procedure, "duration", elapsed)
		}

		return resp, err
	}
}

func (i *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		slog.Info("rpc stream open", "peer", conn.Peer().Addr, "procedure", conn.Spec().Procedure)
		err := next(ctx, conn)
		elapsed := time.Since(start)
		if err != nil {
			slog.Error("rpc stream error", "procedure", conn.Spec().Procedure, "error", err, "duration", elapsed)
		} else {
			slog.Info("rpc stream closed", "procedure", conn.Spec().Procedure, "duration", elapsed)
		}
		return err
	}
}
