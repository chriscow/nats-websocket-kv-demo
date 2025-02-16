# Go-Nuts

Go-Nuts is a Go-based application that demonstrates the use of WebSockets and NATS JetStream for real-time messaging. The application can operate in different modes to handle WebSocket connections and worker events.

## Features

- **WebSocket Server**: Handles WebSocket connections and communicates with clients.
- **NATS JetStream**: Utilizes NATS for message streaming and processing.
- **Worker Mode**: Processes messages from a NATS stream and performs tasks.
- **Health Check Endpoint**: Provides a simple health check endpoint for monitoring.

## Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/go-nuts.git
   cd go-nuts
   ```

2. **Install dependencies**:
   Ensure you have Go installed. Then, run:
   ```bash
   go mod tidy
   ```

3. **Set up environment variables**:
   Create a `.env` file in the root directory and specify the necessary environment variables. For example:
   ```
   NATS_URL=nats://localhost:4222
   PORT=8080
   ```

## Usage

### Running the Application

You can run the application in different modes:

- **WebSocket Mode**: Only handles WebSocket connections.
- **Worker Mode**: Only processes worker events.
- **All Mode**: Handles both WebSocket connections and worker events.

To start the application, use the following command:

```bash
go run main.go -mode=<mode>
```


Replace `<mode>` with `ws`, `worker`, or `all`.

### WebSocket Endpoint

- **URL**: `ws://localhost:<PORT>/ws`
- **Health Check**: `http://localhost:<PORT>/_healthz`

### NATS Configuration

Ensure that a NATS server is running and accessible. The application uses NATS JetStream for message streaming.

### Configuration

- **NATS Server**: Configure the NATS server URL in the `.env` file.
- **HTTP Port**: Set the desired HTTP port in the `.env` file.

## Code Structure

- **main.go**: The entry point of the application, handling initialization and mode selection.
- **types/WorkerEvent.proto**: Protocol buffer definition for worker events.
- **types/WorkerEvent.pb.go**: Generated Go code from the protocol buffer definition.

## Development

### Debugging

The application includes verbose logging to help trace the flow of data and identify issues.

### Testing

Tests are written using the Go standard library. To run tests, use:
```bash
go test ./...
```


## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## Acknowledgments

- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [NATS JetStream](https://github.com/nats-io/nats.go)
- [Google Protocol Buffers](https://developers.google.com/protocol-buffers)
