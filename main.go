package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	types "go-nuts/types"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)



func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize NATS connection
	var natsUrl = os.Getenv("NATS_URL")
	if natsUrl == "" {
		natsUrl = nats.DefaultURL
	}
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal("Error conencting to NATS server: ", err)
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatal("Error cannot create NATS JetStream: ", err)
	}
	sSess, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "session",
		Description: "Stream for client WebSockets",
		Subjects:    []string{"session.*"},
		Retention:   jetstream.WorkQueuePolicy,
	})
	if err != nil {
		log.Fatal("Error cannot creating session stream: ", err)
	}

	// Initialize backend mode
	var handleWS = false
	var handleWorker = false
	switch backendMode := os.Getenv("MODE"); backendMode {
	case "ws":
		log.Println("Running WebSocket mode")
		handleWS = true
	case "worker":
		log.Println("Running worker mode")
		handleWorker = true
	default:
		log.Println("Running all mode")
		handleWS = true
		handleWorker = true
	}

	// Handle worker events
	if handleWorker {
		// Stream
		sWorker, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:        "worker",
			Description: "Stream for worker jobs",
			Subjects:    []string{"worker"},
			Retention:   jetstream.WorkQueuePolicy,
		})
		if err != nil {
			log.Fatal("Error cannot creating worker stream: ", err)
		}

		// Consumer
		consWorker, err := sWorker.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
			Name:      "worker",
			AckPolicy: jetstream.AckExplicitPolicy,
		})
		if err != nil {
			log.Fatal("Error cannot creating worker consumer: ", err)
		}

		// Consumer handler
		cc, err := consWorker.Consume(func(msg jetstream.Msg) {
			WorkerEvent := types.WorkerEvent{}
			if err := proto.Unmarshal(msg.Data(), &WorkerEvent); err != nil {
				log.Fatal("Error unmarshalling message from worker: ", err)
				msg.Nak()
			}
			if err := handleWorkerEvent(ctx, &WorkerEvent, js); err != nil {
				log.Fatal("Error handling worker event: ", err)
				msg.Nak()
			}
			msg.Ack()
		})
		if err != nil {
			log.Fatal("Error consuming to worker stream: ", err)
		}
		defer cc.Stop()

		log.Printf("Subscribed to NATS as worker")
	}

	// Handle WebSocket requests
	if handleWS {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			handleWebSocket(w, r, upgrader, js, sSess)
		})
	}

	// Handle healthcheck requests
	http.HandleFunc("/_healthz", handleHealthz)

	// Initialize HTTP listener
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	log.Printf("Serving HTTP on %v\n", httpPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", httpPort), nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, upgrader *websocket.Upgrader, js jetstream.JetStream, sSess jetstream.Stream) {
	ctx := r.Context()

	// Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading websocket connection: ", err)
		return
	}
	defer conn.Close()

	// Create a session subscription from worker
	session := uuid.NewString()
	consSession, err := sSess.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:           fmt.Sprintf("session-%v", session),
		AckPolicy:      jetstream.AckExplicitPolicy,
		FilterSubjects: []string{fmt.Sprintf("session.%v", session)},
	})
	if err != nil {
		log.Println("Error creating session consumer: ", err)
		return
	}

	cc, err := consSession.Consume(func(msg jetstream.Msg) {
		defer msg.Ack()
		workerEvent := &types.WorkerEvent{}
		err = proto.Unmarshal(msg.Data(), workerEvent)
		if err != nil {
			log.Println("Error unmarshalling message from worker: ", err)
			return
		}
		err := conn.WriteMessage(websocket.TextMessage, []byte(workerEvent.GetContent()))
		if err != nil {
			log.Println("Error writing message to WebSocket: ", err)
			return
		}
	})
	if err != nil {
		log.Println("Error consuming to session stream: ", err)
		return
	}
	defer cc.Stop()

	// Listen for WebSocket message and send it to subscription
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error receiving message from WebSocket: ", err)
			break
		}

		workerEvent := &types.WorkerEvent{
			Session: session,
			Content: string(message),
		}
		weBytes, err := proto.Marshal(workerEvent)
		if err != nil {
			log.Println("Error marshalling message from WebSocket: ", err)
			break
		}

		_, err = js.Publish(ctx, "worker", weBytes)
		if err != nil {
			log.Println("Error publishing message to NATS worker: ", err)
			break
		}
	}
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleWorkerEvent(ctx context.Context, m *types.WorkerEvent, js jetstream.JetStream) error {
	userSession := m.Session
	reply := &types.WorkerEvent{
		Session: userSession,
		Content: fmt.Sprintf("Got: %v", m.Content),
	}
	subject := fmt.Sprintf("session.%v", userSession)
	msgBytes, err := proto.Marshal(reply)
	if err != nil {
		return err
	}
	js.Publish(ctx, subject, msgBytes)
	return nil
}
