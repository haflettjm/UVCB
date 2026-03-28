package core

import (
	"testing"
	"time"

	"github.com/haflettjm/UVCB/internal/config"
	"github.com/haflettjm/UVCB/internal/models"
)

func TestPublishSubscribe(t *testing.T) {
	conf := config.Config{
		NATS: config.NATSConfig{
			URL: "nats://localhost:4222",
		},
	}
	bus, err := NewMessageBus(conf)
	if err != nil {
		t.Fatal(err)
	}

	defer bus.Close()

	received := make(chan models.Message, 1)

	err = bus.Subscribe("test.subject", func(msg models.Message) {
		received <- msg
	})

	if err != nil {
		t.Fatal(err)
	}

	testMsg := models.Message{
		Content:        "hello from test",
		PlatformOrigin: "test",
	}

	err = bus.Publish("test.subject", testMsg)

	select {
	case msg := <-received:
		if msg.Content != "hello from test" {
			t.Errorf("expected 'hello from test'. got '%s'", msg.Content)
		}
	case <-time.After(2 * time.Second):

		t.Fatal("timed out waiting for message")
	}

}
