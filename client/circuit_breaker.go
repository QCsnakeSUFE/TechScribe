package main

import (
	"context"
	"grpc/pb"
	"log"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerOpen
	breakerHalfOpen
)

type circuitBreaker struct {
	mu               sync.Mutex
	state            breakerState
	failures         int
	failureThreshold int
	openedAt         time.Time
	openTimeOut      time.Duration
}

func NewCircuitBreaker(failureThreshold int, openTimeOut time.Duration) *circuitBreaker {
	return &circuitBreaker{
		failureThreshold: failureThreshold,
		openTimeOut:      openTimeOut,
	}
}

func (b *circuitBreaker) unaryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if !isCalculatorUnary(method) {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		if !b.allow() {
			log.Printf("[CircuitBreaker] method=%s state=open action=fallback", method)
			return fallbackCalculator(method, req, reply, status.Error(codes.Unavailable, "circuit breaker is open"))
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			b.recordSuccess()
			return nil
		}

		if isBreakerFailure(err) {
			b.recordFailure()
			log.Printf("[CircuitBreaker] method=%s code=%s action=fallback", method, status.Code(err))
			return fallbackCalculator(method, req, reply, err)
		}

		return err
	}
}

func (b *circuitBreaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state != breakerOpen {
		return true
	}
	if time.Since(b.openedAt) < b.openTimeOut {
		return false
	}

	b.state = breakerHalfOpen
	return true
}

func (b *circuitBreaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = breakerClosed
	b.failures = 0
}

func (b *circuitBreaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures++
	if b.state == breakerHalfOpen || b.failures >= b.failureThreshold {
		b.state = breakerOpen
		b.openedAt = time.Now()
		log.Printf("[CircuitBreaker] state=open failures=%d", b.failures)
	}
}

func isBreakerFailure(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

func isCalculatorUnary(method string) bool {
	return strings.HasPrefix(method, "/calculator.CalculatorService/") && !strings.HasSuffix(method, "/Chat")
}

func fallbackCalculator(method string, req any, reply any, cause error) error {
	calculateReq, ok := req.(*pb.CalculateRequest)
	if !ok {
		return cause
	}

	calculateReply, ok := reply.(*pb.CalculateResponse)
	if !ok {
		return cause
	}

	switch {
	case strings.HasSuffix(method, "/Add"):
		calculateReply.Result = float64(calculateReq.Num1 + calculateReq.Num2)
	case strings.HasSuffix(method, "/Subtract"):
		calculateReply.Result = float64(calculateReq.Num1 - calculateReq.Num2)
	case strings.HasSuffix(method, "/Multiply"):
		calculateReply.Result = float64(calculateReq.Num1 * calculateReq.Num2)
	case strings.HasSuffix(method, "/Divide"):
		if calculateReq.Num2 == 0 {
			return status.Error(codes.InvalidArgument, "num2 cannot be zero")
		}
		calculateReply.Result = float64(calculateReq.Num1) / float64(calculateReq.Num2)
	default:
		return cause
	}

	log.Printf("[Fallback] method=%s result=%g cause=%s", method, calculateReply.Result, status.Code(cause))
	return nil
}
