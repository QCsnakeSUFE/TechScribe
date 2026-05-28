package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func unaryLogInterceptor(
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
	log.Printf(
		"[Unary] method=%s code=%s duration=%s",
		info.FullMethod,
		code,
		time.Since(start),
	)
	return resp, err
}

// to do: refine this function to record the duration time of each message.
func streamLoggingInterceptor(
	srv any,
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	err := handler(srv, stream)
	code := status.Code(err)
	if err == nil {
		code = codes.OK
	}
	log.Printf(
		"[Stream] method=%s code=%s duration=%s client_stream=%t server_stream=%t",
		info.FullMethod,
		code,
		time.Since(start),
		info.IsClientStream,
		info.IsServerStream,
	)
	return err
}
