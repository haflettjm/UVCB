package core

import (
	"encoding/json"

	"github.com/haflettjm/UVCB/internal/config"
	"github.com/haflettjm/UVCB/internal/models"
	"github.com/nats-io/nats.go"
)

type MsgBus struct {
	conn *nats.Conn
}

func NewMessageBus(conf config.Config) (MsgBus, error) {
	connection, err := nats.Connect(conf.NATS.URL)
	if err != nil {
		return MsgBus{}, err
	}
	return MsgBus{conn: connection}, nil
}

func (bus MsgBus) Publish(subject string, msg models.Message) error {
	msgjson, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = bus.conn.Publish(subject, msgjson)
	if err != nil {
		return err
	}
	return nil
}

func (bus MsgBus) Subscribe(subject string, handler func(models.Message)) error {
	_, err := bus.conn.Subscribe(subject, func(msg *nats.Msg) {
		var message models.Message
		err := json.Unmarshal(msg.Data, &message)
		if err != nil {
			return
		}
		handler(message)
	})
	if err != nil {
		return err
	}

	return nil
}
func (bus MsgBus) Close() {
	bus.conn.Close()
}
