# Ladno

Desktop API client for REST, WebSocket, Socket.IO, gRPC, NATS, and Kafka.

## Features

- **Workspaces** with collections, folders, and environments (`{{var}}` in URLs, headers, bodies, and auth)
- **REST** — methods, query/path params, headers, cookies, raw / form-data / urlencoded body, pre/post scripts
- **WebSocket** — connect, send, live message log
- **Socket.IO** — connect, emit, listen by event, handshake auth (token, API key, JSON)
- **gRPC** — import `.proto`, unary calls, metadata, JSON message, scripts
- **NATS** — collection connection, publish / request / subscribe
- **Kafka** — collection brokers/SASL, produce and consume
- **Auth** inherited from collection → folder → request (per protocol)
- **Logs** and stream messages with stick-to-bottom

---

License: [MIT](LICENSE)
