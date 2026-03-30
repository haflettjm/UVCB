package config

import (
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
}

type RedisConfig struct {
	URL      string `yaml:"url"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
}

type NATSConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type RTMPConfig struct {
	StreamURL    string `yaml:"stream_url"`
	StreamKey    string `yaml:"stream_key"`
	VideoBitrate int    `yaml:"video_bitrate"`
	AudioBitrate int    `yaml:"audio_bitrate"`
	ResW         int    `yaml:"res_w"`
	ResH         int    `yaml:"res_h"`
	Framerate    int    `yaml:"framerate"`
	FFmpegPath   string `yaml:"ffmpeg_path"`
	Enabled      bool   `yaml:"enabled"`
}

type OSCConfig struct {
	TargetIP      string `yaml:"target_ip"`
	TargetPort    int    `yaml:"target_port"`
	Enabled       bool   `yaml:"enabled"`
	MessageFormat string `yaml:"message_format"`
}

type SRSConfig struct {
	Address  string `yaml:"address"`
	RTMPPort int    `yaml:"rtmp_port"`
	RTSPPort int    `yaml:"rtsp_port"`
	Enabled  bool   `yaml:"enabled"`
}

type WorldConfig struct {
	WorldID         string   `yaml:"world_id"`
	InstanceID      string   `yaml:"instance_id"`
	StatusAPIPath   string   `yaml:"status_api_path"`
	AllowedWorldIDs []string `yaml:"allowed_world_ids"`
}

type VRChatConfig struct {
	Token   string
	Enabled bool        `yaml:"enabled"`
	RTMP    RTMPConfig  `yaml:"rtmp"`
	OSC     OSCConfig   `yaml:"osc"`
	SRS     SRSConfig   `yaml:"srs"`
	World   WorldConfig `yaml:"world"`
}

type DiscordConfig struct {
	Token           string
	GuildIDs        []uint64 `yaml:"guild_ids"`
	TextChannelIDs  []uint64 `yaml:"text_channel_ids"`
	VoiceChannelIDs []uint64 `yaml:"voice_channel_ids"`
	Enabled         bool     `yaml:"enabled"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	NATS     NATSConfig     `yaml:"nats"`
	VRChat   VRChatConfig   `yaml:"vrchat"`
	Discord  DiscordConfig  `yaml:"discord"`
}

func Load(path string) (Config, error) {
	err := godotenv.Load()
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var conf Config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return Config{}, err
	}
	conf.Discord.Token = os.Getenv("UVCB_DISCORD_TOKEN")
	conf.VRChat.Token = os.Getenv("UVCB_VRCHAT_TOKEN")
	conf.Database.Password = os.Getenv("UVCB_DB_PASSWORD")
	conf.VRChat.RTMP.StreamKey = os.Getenv("UVCB_RTMP_STREAM_KEY")
	conf.Redis.Password = os.Getenv("UVCB_REDIS_PASSWORD")
	conf.NATS.Token = os.Getenv("UVCB_NATS_TOKEN")

	return conf, nil
}
