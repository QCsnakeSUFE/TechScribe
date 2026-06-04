# TechScribe gRPC Playground

TechScribe is a small gRPC learning project that demonstrates production-style RPC building blocks without turning into a full framework.

## What It Covers

- Unary RPC calculator: add, subtract, multiply, divide.
- Bidirectional streaming chat.
- Metadata, request IDs, trace IDs, auth, and interceptors.
- Deadlines, retry policy, circuit breaker, and local fallback.
- Health checks, reflection, graceful shutdown, and basic metrics.
- etcd registration, discovery, resolver watch, and round-robin balancing.

## Quick Start

Start the server:

```bash
go run ./server
```

Run calculator commands:

```bash
GRPC_AUTH_TOKEN=dev-token make add 1 2
GRPC_AUTH_TOKEN=dev-token make sub 8 3
GRPC_AUTH_TOKEN=dev-token make mul 6 7
GRPC_AUTH_TOKEN=dev-token make div 36 6
```

Run streaming chat:

```bash
GRPC_AUTH_TOKEN=dev-token make chat
```

Check health:

```bash
make health
```

View metrics:

```bash
curl http://localhost:9090/debug/vars
```

Use a custom metrics port:

```bash
METRICS_ADDR=:19090 go run ./server
curl http://localhost:19090/debug/vars
```

Disable metrics:

```bash
METRICS_ADDR=off go run ./server
```

## etcd Discovery

Start etcd locally first, then run a server that registers itself:

```bash
ENABLE_ETCD_REGISTRY=true GRPC_SERVER_ADDR=:51061 go run ./server
```

Use the custom etcd resolver from the client:

```bash
ENABLE_ETCD_RESOLVER=true GRPC_AUTH_TOKEN=dev-token make radd 1 2
ENABLE_ETCD_RESOLVER=true GRPC_AUTH_TOKEN=dev-token make rchat
```

## Useful Environment Variables

- `GRPC_SERVER_ADDR`: server listen address, default `:50001`.
- `GRPC_REGISTER_ADDR`: address registered into etcd.
- `GRPC_TARGET_ADDR`: direct client target, default `localhost:50001`.
- `GRPC_AUTH_TOKEN`: set to `dev-token` for protected calculator/chat RPCs.
- `ENABLE_ETCD_REGISTRY=true`: enable server registration into etcd.
- `ENABLE_ETCD_DISCOVERY=true`: query etcd once before dialing.
- `ENABLE_ETCD_RESOLVER=true`: use the custom watch-based etcd resolver.
- `METRICS_ADDR`: metrics HTTP address, default `:9090`; use `off` to disable.

## Direction

The current calculator and chat APIs are learning surfaces. The long-term direction is a full-stack AI conversation system where browser-facing HTTP/WebSocket/SSE APIs call internal gRPC services for chat, ASR, TTS, and provider integration.
