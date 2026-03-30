package bridge

import (
	"context"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/haflettjm/UVCB/internal/config"
	"github.com/haflettjm/UVCB/internal/core"
	"github.com/haflettjm/UVCB/internal/models"
	"go.uber.org/zap"
)

type DiscordBridge struct {
	channels         map[string]string
	ActiveTextUsers  map[string]string
	ActiveVoiceUsers map[string]string
	bus              *core.MsgBus
	client           *bot.Client
	logger           *zap.Logger
}

func NewDiscordBridge(cfg config.DiscordConfig, bus *core.MsgBus, log *zap.Logger) (*DiscordBridge, error) {
	dclient, err := disgo.New(cfg.Token,
		bot.WithGatewayConfigOpts(gateway.WithIntents(
			gateway.IntentGuildMessages,
			gateway.IntentMessageContent,
		)),
		bot.WithEventListenerFunc(func(e *events.MessageCreate) {
			msg := models.Message{
				Content:        e.Message.Content,
				UserName:       e.Message.Author.Username,
				UserID:         string(e.Message.Author.ID),
				Time:           e.MessageID.Time(),
				Type:           "text",
				MessageType:    "message",
				PlatformOrigin: "discord",
			}
			err := bus.Publish("bridge.text.discord", msg)
			if err != nil {
				log.Error("failed to publish discord message", zap.Error(err))
			}
		}),
	)
	if err != nil {
		return nil, err
	}
	return &DiscordBridge{
		client:           dclient,
		bus:              bus,
		channels:         make(map[string]string),
		ActiveTextUsers:  make(map[string]string),
		ActiveVoiceUsers: make(map[string]string),
		logger:           log,
	}, nil
}
func (dbridge *DiscordBridge) Connect() error {
	return dbridge.client.OpenGateway(context.TODO())
}

func (dbridge *DiscordBridge) Disconnect() error {
	dbridge.client.Close(context.TODO())
	return nil
}

func (dbridge *DiscordBridge) SendMessage(msg models.Message) error {
	return nil
}

func (dbridge *DiscordBridge) ReceiveMessages() (<-chan models.Message, <-chan error) {
	return nil, nil
}

func (dbridge *DiscordBridge) Platform() string {
	return "discord"
}

func (dbridge *DiscordBridge) Status() string {
	return dbridge.client.Gateway.Status().String()
}
