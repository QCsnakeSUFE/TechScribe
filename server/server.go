package main

import (
	"context"
	"fmt"
	"grpc/pb"
	"io"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

// can be changed later
func generateReply(msg string) string {
	return fmt.Sprintf("Robot reply: I heard you saying '%s'", msg)
}

func main() {
	lis, err := net.Listen("tcp", ":50001")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.UnaryInterceptor(unaryLoggingInterceptor),
		grpc.StreamInterceptor(streamLoggingInterceptor),
	)
	pb.RegisterCalculatorServiceServer(s, &server{})
	reflection.Register(s)
	log.Println("gRPC server is running on port :50001..")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
