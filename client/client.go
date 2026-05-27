package main

import (
	"bufio"
	"context"
	"fmt"
	"grpc/pb"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	conn, err := grpc.NewClient(":50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
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

	runCalculator(c, op)
}

func runCalculator(c pb.CalculatorServiceClient, op string) {
	if len(os.Args) != 4 {
		log.Fatalf("usage: go run ./client <add|sub|mul|div> <num1> <num2>")
	}

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
		res, err = c.Add(context.Background(), req)
	case "sub":
		res, err = c.Subtract(context.Background(), req)
	case "mul":
		res, err = c.Multiply(context.Background(), req)
	case "div":
		res, err = c.Divide(context.Background(), req)
	default:
		log.Fatalf("unknown operation %q; use add, sub, mul, or div", op)
	}
	if err != nil {
		log.Fatalf("%s failed: %v", op, err)
	}

	fmt.Printf("%s result: %g\n", op, res.Result)
}

func runChat(c pb.CalculatorServiceClient) {
	stream, err := c.Chat(context.Background())
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
			fmt.Println("server disconnected, please restart the server and try again")
			return
		}

		resp, err := stream.Recv()
		if err != nil {
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
