package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/haflettjm/UVCB/internal/bridge"
	"github.com/haflettjm/UVCB/internal/config"
	"github.com/haflettjm/UVCB/internal/core"
	"github.com/haflettjm/UVCB/internal/models"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Error Initializing Logger: ", err)
	}
	logger.Info("Server Starting...")
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		logger.Fatal("Failed to load Config Files", zap.Error(err))
	}
	bus, err := core.NewMessageBus(cfg, logger)
	if err != nil {
		logger.Panic("Message bus failed to Initialize", zap.Error(err))
	}
	logger.Info(cfg.Discord.Token)
	bus.Subscribe("bridge.text.>", func(msg models.Message) {
		logger.Info("mesesage received",
			zap.String("user", msg.UserName),
			zap.String("content", msg.Content),
			zap.String("platform", msg.PlatformOrigin),
		)
	})

	dbridge, err := bridge.NewDiscordBridge(cfg.Discord, &bus, logger)
	if err != nil {
		logger.Panic("Discord Bridge failed to Initialize", zap.Error(err))
	}

	err = dbridge.Connect()
	if err != nil {
		logger.Panic("Discord Bridge failed to Conncet", zap.Error(err))
	}
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
	logger.Info("Shutting Down...")
	dbridge.Disconnect()
	bus.Close()
}
