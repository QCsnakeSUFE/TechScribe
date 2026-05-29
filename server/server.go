package main

import (
	"context"
	"fmt"
	"grpc/pb"
	"io"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedCalculatorServiceServer
}

func (s *server) Add(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	log.Printf("Received Add request: num1=%d, num2=%d", req.Num1, req.Num2)
	return &pb.CalculateResponse{Result: float64(req.Num1 + req.Num2)}, nil
}

func (s *server) Subtract(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	log.Printf("Received Subtract request: num1=%d, num2=%d", req.Num1, req.Num2)
	return &pb.CalculateResponse{Result: float64(req.Num1 - req.Num2)}, nil
}

func (s *server) Multiply(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	log.Printf("Received Multiply request: num1=%d, num2=%d", req.Num1, req.Num2)
	return &pb.CalculateResponse{Result: float64(req.Num1 * req.Num2)}, nil
}

func (s *server) Divide(ctx context.Context, req *pb.CalculateRequest) (*pb.CalculateResponse, error) {
	log.Printf("Received Divide request: num1=%d, num2=%d", req.Num1, req.Num2)
	if req.Num2 == 0 {
		return nil, status.Error(codes.InvalidArgument, "num2 cannot be zero")
	}
	return &pb.CalculateResponse{Result: float64(req.Num1) / float64(req.Num2)}, nil
}

func (s *server) Chat(stream pb.CalculatorService_ChatServer) error {
	log.Println("New Chat stream established.")
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client closed the stream.")
			return nil
		}
		if err != nil {
			log.Printf("Failed to receive from stream: %v", err)
			return err
		}
		log.Printf("Received message: %s", req.Message)
		reply := generateReply(req.Message)
		resp := &pb.ChatResponse{Reply: reply}
		if err := stream.Send(resp); err != nil {
			log.Printf("Failed to send to stream: %v", err)
			return err
		}
	}
}

type loggingServerStream struct {
	grpc.ServerStream
	method    string
	recvCount int
	sendCount int
}

func (s *loggingServerStream) RecvMsg(m any) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		s.recvCount++
		log.Printf("[Stream Recv] method=%v type=%T recv_count=%d", s.method, m, s.recvCount)
	}
	return err
}

func (s *loggingServerStream) SendMsg(m any) error {
	err := s.ServerStream.SendMsg(m)
	if err == nil {
		s.sendCount++
		log.Printf("[Stream Send] method=%s type=%T send_count=%d", s.method, m, s.sendCount)
	}
	return err
}

func generateReply(msg string) string {
	return fmt.Sprintf("Robot reply: I heard you saying '%s'", msg)
}

func main() {
	addr := os.Getenv("GRPC_SERVER_ADDR")
	if addr == "" {
		addr = ":50001"
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			unaryLoggingInterceptor,
			unaryAuthInterceptor,
		),
		grpc.ChainStreamInterceptor(
			streamLoggingInterceptor,
			streamAuthInterceptor,
		),
	)

	pb.RegisterCalculatorServiceServer(s, &server{})

	healthServer := health.NewServer()
	healthServer.SetServingStatus(
		"calculator.CalculatorService",
		healthgrpc.HealthCheckResponse_SERVING,
	)
	healthgrpc.RegisterHealthServer(s, healthServer)

	reflection.Register(s)
	log.Printf("gRPC server is running on port %s...", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
