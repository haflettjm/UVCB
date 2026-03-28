# UVCB — Universal Virtual Chat Bridge

Bridge voice, video, and text across Discord, VRChat, and more — from a single self-hosted Go binary.

Online communities are fragmented across platforms. Your Discord server, VRChat world, Mumble channel, and Matrix room each hold part of the conversation. UVCB bridges them together — relaying voice, video, and text in real time so nobody has to switch apps to stay connected. It's open source, self-hostable, and built for communities that refuse to live on just one platform.

---

## Supported Platforms

| Platform | Protocol | Voice | Video | Text | Phase |
|----------|----------|-------|-------|------|-------|
| Discord | Bot API | Yes | Yes | Yes | 1 (MVP) |
| VRChat | RTMP/RTSP + OSC | Yes | Yes (overlay) | Yes | 1 (MVP) |
| Mumble | Murmur | Yes | No | Yes | 2 |
| Matrix | WebRTC (Element Call) | Yes | Yes | Yes | 2 |
| IRC | IRC protocol | No | No | Yes | 2 |
| XMPP | Jingle | Yes | Yes | Yes | 3 |
| QQ | go-cqhttp / SILK | Yes | No | Yes | 3 |

---

## Architecture

UVCB uses a **selective MCU** architecture. Rather than transcoding every stream (expensive) or blindly forwarding packets (incompatible across codecs), it takes the smart middle path: platforms that share the same codec get direct passthrough, and transcoding only happens at codec boundaries.

Four of six voice platforms (Discord, Mumble, Matrix, XMPP) speak **Opus at 48 kHz** natively. Audio between them passes through with zero transcoding. Only the VRChat/RTMP boundary (requires AAC) and QQ (requires SILK) need dedicated transcoding via FFmpeg.

### Data Flow

The system runs three pipelines as goroutines:

**Audio pipeline** — Each platform connector decodes incoming audio to PCM 48 kHz, or forwards raw Opus packets when source and destination codecs match. A central mixer sums PCM samples from all active sources. Per-output encoders produce Opus for Discord/Mumble/Matrix/XMPP and pipe PCM to FFmpeg for AAC encoding on the RTMP output.

**Video/overlay pipeline** — A Go canvas renderer produces chat overlay frames at 30 fps. These pipe to FFmpeg along with the mixed audio for H.264 + AAC RTMP output. When bridging participant video, H.264 frames pass through without re-encoding (negotiated via SDP).

**Text pipeline** — Messages from all platforms flow through a central message bus (NATS). Each connector publishes and subscribes. Text feeds the chat overlay renderer and the web dashboard via WebSocket.

### Codec Compatibility

| Source / Dest | Discord | Mumble | Matrix | XMPP | RTMP/VRChat | QQ |
|---|---|---|---|---|---|---|
| Discord (Opus 48k stereo) | Direct | Passthrough* | Direct | Direct** | Transcode (AAC) | Transcode (SILK) |
| Mumble (Opus 48k mono) | Passthrough* | Direct | Passthrough* | Direct** | Transcode | Transcode |
| Matrix (Opus 48k stereo) | Direct | Passthrough* | Direct | Direct** | Transcode | Transcode |
| RTMP (AAC 44.1/48k) | Transcode | Transcode | Transcode | Transcode | Direct | Transcode |
| QQ (SILK 24k) | Transcode | Transcode | Transcode | Transcode | Transcode | Direct |

\* = mono/stereo channel adjustment only
\** = Opus negotiable, may fall back to G.711

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go |
| Web Framework | Gin |
| Database | PostgreSQL (pgx) |
| Cache | Redis (go-redis) |
| Message Bus | NATS |
| Logging | Zap |
| Metrics | Prometheus |
| Frontend | Nuxt + Vue.js |
| Deployment | Pulumi |

---

## Library Stack

### Media and Protocol Libraries

| Component | Library | Pure Go | Role |
|-----------|---------|---------|------|
| WebRTC transport | `pion/webrtc` v4 | Yes | Matrix/Element Call, XMPP Jingle connections |
| RTP/RTCP | `pion/rtp`, `pion/rtcp` | Yes | Packet parsing, payloading/depayloading |
| Opus decode | `pion/opus` | Yes | Decode Opus to PCM without CGo |
| Opus encode/decode | `hraban/opus` (WASM/wazero) | Yes (via WASM) | Full encode/decode without CGo |
| RTMP protocol | `yutopp/go-rtmp` | Yes | RTMP server/client for VRChat pipeline |
| Discord API | `disgoorg/disgo` | Yes | Discord bot and voice connection |
| Mumble client | `stieneee/gumble` | Yes (core) | Mumble protocol client |
| Chat overlay | `fogleman/gg` | Yes | Render chat UI as image frames for video |
| TURN server | `pion/turn` v5 | Yes | Embedded NAT traversal |
| Transcoding | FFmpeg (subprocess) | N/A | Opus to AAC, frame encoding, RTMP mux |
| RTMP relay | SRS v7 (sidecar) | N/A | RTSP/HLS/MPEG-TS delivery to VRChat |

### Infrastructure Libraries

| Component | Library | Role |
|-----------|---------|------|
| HTTP router | `gin-gonic/gin` | REST API and dashboard backend |
| PostgreSQL | `jackc/pgx` | Connection pooling, queries, migrations |
| Redis | `redis/go-redis` | Caching, session state, pub/sub |
| Message bus | `nats-io/nats.go` | Inter-service messaging, event routing |
| Logging | `uber-go/zap` | Structured logging |
| Metrics | `prometheus/client_golang` | Bridge health, latency, stream metrics |

Video codec strategy: negotiate **H.264 everywhere** via SDP to avoid video transcoding entirely.

---

## VRChat Integration

VRChat receives the bridge stream through its **AVPro Video Player**:

- **PC**: RTSP for lowest latency (~1-3 seconds)
- **Quest/Android**: MPEG-TS or HLS (HTTPS required)
- Stream carries both mixed audio and chat overlay video in a single RTMP/RTSP stream
- Recommended format: H.264 Baseline/Main @ 720p, AAC stereo 128 kbps, 2500-4000 kbps total

**Constraints**: VRChat's URL allowlist requires either enabling "Allow Untrusted URLs" or world-creator allowlisting of the bridge domain (max 10 domains per world). Udon cannot construct URLs at runtime — the stream endpoint is hardcoded at world build time.

**Text**: The OSC chatbox (`/chatbox/input`, UDP 9000) supports 144 ASCII characters at ~1.5s update rate — useful for status messages, not full chat relay.

**Overlay alternative**: For users not in the custom bridge world, OVR Toolkit or Desktop+ can display the Nuxt dashboard as a VR overlay.

---

## Roadmap

### Phase 1 — MVP: Discord + VRChat Bridge
- Discord voice, video, and text via bot API
- VRChat RTMP/RTSP output with chat overlay and mixed audio
- Text bridging with chat overlay rendering
- FFmpeg subprocess for AAC encoding and RTMP mux
- SRS sidecar for stream delivery
- Nuxt web dashboard for status and admin
- PostgreSQL for persistent state, Redis for caching, NATS for message routing

### Phase 2 — Expand Platform Support
- Mumble voice and text via gumble
- Matrix voice, video, and text via Pion WebRTC
- IRC text relay

### Phase 3 — Niche Platforms
- XMPP Jingle voice and video
- QQ voice via SILK transcoding (CGo required)

### Phase 4 — Polish and Optimization
- Hardware-accelerated encoding (NVENC, VAAPI, QSV)
- HLS output for Quest compatibility
- Adaptive quality based on CPU load
- Recording and playback

---

## Hardware Requirements

| Tier | Specs | Capability |
|------|-------|------------|
| Budget VPS | 2 cores, 2 GB RAM | Audio bridge + basic RTMP output |
| Recommended | 4 cores, 8 GB RAM | Full audio + video bridge, ~20-30 participants |
| Heavy use | 8 cores, 16 GB RAM | Multiple rooms, video transcoding, recording |

Bandwidth: ~100 kbps per voice participant, 3-6 Mbps for RTMP output at 720p, ~500 kbps per video stream at 360p.

---

## Deployment

UVCB ships as a **single Go binary** with an optional Docker Compose file. FFmpeg is a required dependency for RTMP output. A built-in TURN server (`pion/turn`) eliminates the need for separate coturn deployment.

Configuration uses YAML with environment variable overrides. Minimal config: a domain name and an API secret.

Recommended setup: Caddy as a reverse proxy for automatic TLS via Let's Encrypt. Infrastructure managed via Pulumi.

---

## Project Structure (Planned)

```
uvcb/
├── cmd/                  # Entry points
├── internal/
│   ├── bridge/           # Per-platform connectors
│   ├── core/             # Message bus, audio mixer, routing
│   └── config/           # Config loading and schema
├── web/                  # Nuxt frontend (dashboard)
├── deployments/
│   ├── k8s/
│   └── compose/
├── scripts/              # CI/CD and helper scripts
└── README.md
```

---

## Getting Started

> Coming soon — the project is in early development.

### Prerequisites
- Go 1.22+
- FFmpeg
- Docker (optional, for SRS relay)
- PostgreSQL
- Redis
- NATS

---

## Contributing

Contributions are welcome. Each platform bridge lives in its own package under `internal/bridge/` and implements a shared interface. See [RESEARCH.md](RESEARCH.md) for deep technical context on architecture decisions and library choices.

---

## License

This project is licensed under the [GNU Affero General Public License v3.0 (AGPL-3.0)](LICENSE).

---

Built by Edji-Ideas LLC. Maintained by Jacob Haflett.
