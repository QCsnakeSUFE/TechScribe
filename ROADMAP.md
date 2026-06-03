# gRPC LLM Chat Roadmap

This project starts as a small gRPC learning project and can evolve into a full-stack AI conversation system.

## Current Scope

- Unary gRPC calculator APIs for add, subtract, multiply, and divide.
- Bidirectional streaming gRPC chat between a terminal client and server.
- Makefile shortcuts for calculator and chat commands.
- Basic stream disconnect handling in the terminal client.
- Unary and stream interceptors for logging, request metadata, and simple auth.
- Standard gRPC health check service.
- etcd-based service registration, discovery, resolver watch, and round-robin balancing.
- Graceful shutdown with health status changes and etcd deregistration.
- Client retry policy, circuit breaker, and local calculator fallback.
- Basic observability with structured log fields, trace IDs, and expvar metrics.

## Project Direction

The long-term direction is to turn this into a gRPC-based AI service layer that can support terminal, web, and possibly mobile clients.

The recommended architecture is:

```text
Web / Mobile / Terminal Client
        |
HTTP JSON / WebSocket / SSE
        |
Backend Gateway
        |
gRPC Internal Services
        |
LLM / ASR / TTS / Cloud Provider APIs
```

In this model, browsers do not need to call gRPC directly. The frontend can use HTTP, WebSocket, or SSE, while backend services use gRPC for strongly typed internal communication.

## Possible Services

### Chat Service

- `Chat`: regular request-response chat.
- `StreamChat`: streaming LLM response.
- Multi-turn conversation context.
- System prompt configuration.
- Model/provider selection.

### Speech Service

- `Transcribe`: upload audio and return text.
- `StreamTranscribe`: stream microphone audio and return partial recognition results.
- Cloud ASR provider integration.

### TTS Service

- `Synthesize`: convert text to speech.
- `StreamSynthesize`: stream generated audio.
- Cloud TTS provider integration.

## Evolution Plan

1. Keep the calculator APIs as stable unary RPC examples.
2. Keep terminal chat as the bidirectional streaming RPC example.
3. Extract the current fake chat reply into a dedicated reply handler.
4. Replace the fake reply handler with a non-streaming LLM SDK call.
5. Upgrade the server to stream LLM chunks through `stream.Send`.
6. Add conversation history and basic session management.
7. Add a web gateway that exposes HTTP/WebSocket/SSE APIs to browsers.
8. Add browser audio recording and connect it to an ASR service.
9. Add TTS output for voice replies.
10. Move provider keys and runtime settings into environment variables or config files.
11. Replace the current expvar metrics and trace ID placeholder with Prometheus and OpenTelemetry when the project has a long-running backend.
12. Add focused tests and deployment documentation when the system shape stabilizes.

## Learned gRPC Capabilities

- Unary RPC and bidirectional streaming RPC.
- Protobuf service and message generation.
- Client metadata, request IDs, and trace IDs.
- Unary and stream interceptors.
- Auth checks in interceptors.
- Health checks and reflection.
- Deadlines and cancellation.
- Retry policy through gRPC service config.
- Client-side circuit breaker and fallback.
- etcd service registration, lease keepalive, deregistration, discovery, resolver watch, and load balancing.
- Graceful shutdown and drain flow.
- Basic server-side logs and metrics.

## Notes

- gRPC is worth keeping as the internal service protocol.
- HTTP JSON, WebSocket, or SSE is still the better fit for browser-facing APIs.
- API keys, local environment files, credentials, and private notes should never be committed.
