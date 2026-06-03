package main

import (
	"context"
	"fmt"
	"grpc/pb"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	var deregister func(context.Context) error
	if os.Getenv("ENABLE_ETCD_REGISTRY") == "true" {
		registerAddr := os.Getenv("GRPC_REGISTER_ADDR")
		if registerAddr == "" {
			registerAddr = "localhost" + addr
		}

		deregister, err = registerService(
			context.Background(),
			[]string{"localhost:2379"},
			"calculator",
			registerAddr,
		)
		if err != nil {
			log.Fatalf("failed to register service: %v", err)
		}
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

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.Serve(lis)
	}()

	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case <-shutdownCtx.Done():
		log.Println("shutdown signal received")
	case err := <-serveErr:
		if err != nil && err != grpc.ErrServerStopped {
			log.Printf("failed to serve: %v", err)
		}
	}

	healthServer.SetServingStatus(
		"calculator.CalculatorService",
		healthgrpc.HealthCheckResponse_NOT_SERVING,
	)
	log.Println("health status set to NOT_SERVING")
	log.Printf("Draining for 2s before deregistering...")
	time.Sleep(2 * time.Second)

	log.Println("shutting down server...")
	if deregister != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := deregister(ctx); err != nil {
			log.Printf("failed to deregister service: %v", err)
		}
	}

	gracefulStopWithTimeout(s, 5*time.Second)
}

func gracefulStopWithTimeout(s *grpc.Server, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("gRPC server stopped gracefully")
	case <-time.After(timeout):
		log.Printf("graceful stop timed out after %s; forcing stop", timeout)
		s.Stop()
	}
}
