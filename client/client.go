package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"grpc/pb"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const defaultServiceConfig = `{
	"loadBalancingPolicy": "round_robin",
	"methodConfig":[
		{
			"name": [
				{
					"service": "calculator.CalculatorService"
				}
			],
			"retryPolicy": {
				"maxAttempts": 4,
				"initialBackoff": "0.1s",
				"maxBackoff": "1s",
				"backoffMultiplier": 2.0,
				"retryableStatusCodes": ["UNAVAILABLE"]
			}
		}
	]
}`

func main() {
	var err error
	addr := os.Getenv("GRPC_TARGET_ADDR")
	if addr == "" && os.Getenv("ENABLE_ETCD_RESOLVER") == "true" {
		registerEtcdResolver([]string{"localhost:2379"})
		addr = "etcd:///calculator"
	} else if addr == "" && os.Getenv("ENABLE_ETCD_DISCOVERY") == "true" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		addr, err = discoverService(ctx, []string{"localhost:2379"}, "calculator")
		if err != nil {
			log.Fatalf("discover service failed: %v", err)
		}
	}
	if addr == "" {
		addr = "localhost:50001"
	}
	breaker := NewCircuitBreaker(2, 5*time.Second)
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(defaultServiceConfig),
		grpc.WithChainUnaryInterceptor(breaker.unaryInterceptor()),
	)
	if err != nil {
		log.Fatalf("connection failed: %v", err)
	}
	defer conn.Close()

	c := pb.NewCalculatorServiceClient(conn)

	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./client <add|sub|mul|div|chat> [num1] [num2]")
	}

	op := os.Args[1]
	if op == "chat" {
		runChat(c)
		return
	}

	if op == "health" {
		runHealth(conn)
		return
	}

	runCalculator(c, op)
}

func runCalculator(c pb.CalculatorServiceClient, op string) {
	if len(os.Args) != 4 {
		log.Fatalf("usage: go run ./client <add|sub|mul|div> <num1> <num2>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, newMetadata())

	num1, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("invalid num1: %v", err)
	}

	num2, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatalf("invalid num2: %v", err)
	}

	req := &pb.CalculateRequest{
		Num1: int32(num1),
		Num2: int32(num2),
	}

	var res *pb.CalculateResponse
	switch op {
	case "add":
		res, err = c.Add(ctx, req)
	case "sub":
		res, err = c.Subtract(ctx, req)
	case "mul":
		res, err = c.Multiply(ctx, req)
	case "div":
		res, err = c.Divide(ctx, req)
	default:
		log.Fatalf("unknown operation %q; use add, sub, mul, or div", op)
	}
	if err != nil {
		log.Fatalf("%s failed: %v", op, err)
	}

	fmt.Printf("%s result: %g\n", op, res.Result)
}

func runHealth(conn *grpc.ClientConn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, newMetadata())

	healthClient := healthgrpc.NewHealthClient(conn)
	res, err := healthClient.Check(ctx, &healthgrpc.HealthCheckRequest{
		Service: "calculator.CalculatorService",
	})
	if err != nil {
		if isUnauthenticated(err) {
			fmt.Println("authentication failed, please set GRPC_AUTH_TOKEN=dev-token and try again")
			return
		}
		log.Fatalf("health check failed: %v", err)
	}
	fmt.Printf("health status: %s\n", res.Status)
}

func runChat(c pb.CalculatorServiceClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, newMetadata())

	stream, err := c.Chat(ctx)
	if err != nil {
		log.Fatalf("chat failed: %v", err)
	}

	fmt.Println("type a message and press Enter; type /quit to exit")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">client: ")
		if !scanner.Scan() {
			break
		}

		message := strings.TrimSpace(scanner.Text())
		if message == "/quit" {
			break
		}
		if message == "" {
			continue
		}

		if err := stream.Send(&pb.ChatRequest{Message: message}); err != nil {
			if err == io.EOF {
				_, err = stream.Recv()
			}
			if isUnauthenticated(err) {
				fmt.Println("authentication failed, please set GRPC_AUTH_TOKEN=dev-token and try again")
				return
			}
			if isServerDisconnected(err) {
				fmt.Println("server disconnected, please restart the server and try again")
				return
			}
			log.Fatalf("send failed: %v", err)
		}

		resp, err := stream.Recv()
		if err != nil {
			if isUnauthenticated(err) {
				fmt.Println("authentication failed, please set GRPC_AUTH_TOKEN=dev-token and try again")
				return
			}
			if isServerDisconnected(err) {
				fmt.Println("server disconnected, please restart the server and try again")
				return
			}
			log.Fatalf("receive failed: %v", err)
		}
		fmt.Printf("server: %s\n", resp.Reply)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("read stdin failed: %v", err)
	}

	if err := stream.CloseSend(); err != nil {
		if !isServerDisconnected(err) {
			log.Fatalf("close send failed: %v", err)
		}
	}
}

func isServerDisconnected(err error) bool {
	if err == io.EOF {
		return true
	}

	code := status.Code(err)
	return code == codes.Unavailable || code == codes.Canceled
}

func isUnauthenticated(err error) bool {
	return status.Code(err) == codes.Unauthenticated
}

func newMetadata() metadata.MD {
	pairs := []string{
		"x-request-id", newRequestID(),
	}

	if token := os.Getenv("GRPC_AUTH_TOKEN"); token != "" {
		pairs = append(pairs, "authorization", "Bearer "+token)
	}

	return metadata.Pairs(pairs...)
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return "req-" + hex.EncodeToString(b[:])
}
