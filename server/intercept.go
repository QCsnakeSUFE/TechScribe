package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func unaryLoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	code := status.Code(err)
	if err == nil {
		code = codes.OK
	}
	requestID, hasAuth := requestInfo(ctx)
	log.Printf(
		"[Unary] method=%s code=%s duration=%s request_id=%s auth=%t",
		info.FullMethod,
		code,
		time.Since(start),
		requestID,
		hasAuth,
	)
	return resp, err
}

func unaryAuthInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is missing")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization token is missing")
	}
	if tokens[0] != "Bearer dev-token" {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization token")
	}
	return handler(ctx, req)
}

func streamLoggingInterceptor(
	srv any,
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	wrapped := &loggingServerStream{
		ServerStream: stream,
		method:       info.FullMethod,
	}
	err := handler(srv, wrapped)
	code := status.Code(err)
	if err == nil {
		code = codes.OK
	}
	requestID, hasAuth := requestInfo(stream.Context())
	log.Printf(
		"[Stream] method=%s code=%s duration=%s client_stream=%t server_stream=%t request_id=%s auth=%t recv_count=%d send_count=%d",
		info.FullMethod,
		code,
		time.Since(start),
		info.IsClientStream,
		info.IsServerStream,
		requestID,
		hasAuth,
		wrapped.recvCount,
		wrapped.sendCount,
	)
	return err
}

func streamAuthInterceptor(
	srv any,
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.Unauthenticated, "metadata is missing")
	}
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return status.Error(codes.Unauthenticated, "authorization token is missing")
	}
	if tokens[0] != "Bearer dev-token" {
		return status.Error(codes.Unauthenticated, "invalid authorization token")
	}
	return handler(srv, stream)
}

func requestInfo(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "-", false
	}

	requestID := "-"
	if values := md.Get("x-request-id"); len(values) > 0 {
		requestID = values[0]
	}

	return requestID, len(md.Get("authorization")) > 0
}
