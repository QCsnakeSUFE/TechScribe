# gRPC LLM Chat Roadmap

This project starts as a small gRPC learning project and can evolve into a full-stack AI conversation system.

## Current Scope

- Unary gRPC calculator APIs for add, subtract, multiply, and divide.
- Bidirectional streaming gRPC chat between a terminal client and server.
- Makefile shortcuts for calculator and chat commands.
- Basic stream disconnect handling in the terminal client.

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

1. Keep the calculator APIs as unary RPC examples.
2. Extract the current chat reply logic into a dedicated reply handler.
3. Replace the fake reply handler with a non-streaming LLM SDK call.
4. Upgrade the server to stream LLM chunks through `stream.Send`.
5. Add conversation history and basic session management.
6. Add a web gateway that exposes HTTP/WebSocket/SSE APIs to browsers.
7. Add browser audio recording and connect it to an ASR service.
8. Add TTS output for voice replies.
9. Move provider keys and runtime settings into environment variables or config files.
10. Add tests, logging, error handling, and deployment documentation.

## Notes

- gRPC is worth keeping as the internal service protocol.
- HTTP JSON, WebSocket, or SSE is still the better fit for browser-facing APIs.
- API keys, local environment files, credentials, and private notes should never be committed.
