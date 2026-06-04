package main

import (
	"context"
	"expvar"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc/codes"
)

var (
	grpcServerRPCStarted          = expvar.NewMap("grpc_server_rpc_started_total")
	grpcServerRPCCompleted        = expvar.NewMap("grpc_server_rpc_completed_total")
	grpcServerRPCErrors           = expvar.NewMap("grpc_server_rpc_errors_total")
	grpcServerRPCLatencyMicros    = expvar.NewMap("grpc_server_rpc_latency_micros_total")
	grpcServerStreamRecvMessages  = expvar.NewMap("grpc_server_stream_recv_messages_total")
	grpcServerStreamSentMessages  = expvar.NewMap("grpc_server_stream_sent_messages_total")
	grpcServerActiveStreams       = expvar.NewInt("grpc_server_active_streams")
	grpcServerActiveMetricsServer = expvar.NewString("grpc_server_metrics_addr")
)

func startMetricsServer() *http.Server {
	addr := os.Getenv("METRICS_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	if addr == "off" {
		log.Println("metrics server disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/debug/vars", expvar.Handler())
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	grpcServerActiveMetricsServer.Set(addr)
	go func() {
		log.Printf("metrics server is running on %s", metricsURL(addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server failed: %v", err)
		}
	}()

	return server
}

func metricsURL(addr string) string {
	host := addr
	if host == "" || host[0] == ':' {
		host = "localhost" + host
	}
	return "http://" + host + "/debug/vars"
}

func shutdownMetricsServer(server *http.Server) {
	if server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("failed to shutdown metrics server: %v", err)
	}
}

func recordRPCStarted(method string) {
	grpcServerRPCStarted.Add(method, 1)
}

func recordRPCCompleted(method string, code codes.Code, duration time.Duration) {
	key := method + "|" + code.String()
	grpcServerRPCCompleted.Add(key, 1)
	grpcServerRPCLatencyMicros.Add(method, duration.Microseconds())
	if code != codes.OK {
		grpcServerRPCErrors.Add(key, 1)
	}
}

func recordStreamMessages(method string, recvCount, sendCount int) {
	if recvCount > 0 {
		grpcServerStreamRecvMessages.Add(method, int64(recvCount))
	}
	if sendCount > 0 {
		grpcServerStreamSentMessages.Add(method, int64(sendCount))
	}
}
