package models

import (
	"io"
	"time"
)

type Attachment struct {
	FileName string
	URI      string
	MIMEType string
}

type Message struct {
	ID               string       // Messages actual id
	ChannelID        string       // Channel or Vrchat "Room" it was sent in
	UserID           string       // User who sent the message
	UserName         string       // Name of the user
	Mentions         []string     // Mentions of the specified users
	Content          string       // Content of the actual message data here
	Time             time.Time    // Time the message was sent
	Attachments      []Attachment // Images, gifs, etc should go here.
	Type             string       // If its a voice message or text message etc
	MessageType      string       // To denote if its a command we don't need to propagate those
	MessageReference string       // In case it can be tagged into reference of something
	PlatformOrigin   string       // Platform of origin
	OriginServerID   string       // WorldID or guildID or ServerID
	Metadata         map[string]string
}

type Bridge interface {
	Connect() error
	Disconnect() error
	SendMessage(msg Message) error
	ReceiveMessages() (<-chan Message, <-chan error)
	Platform() string
	Status() string
}

type AudioCapable interface {
	AudioReader() io.ReadCloser
	AudioWriter() io.WriteCloser
}

type VideoCapable interface {
	VideoReader() io.ReadCloser
	VideoWriter() io.WriteCloser
}
