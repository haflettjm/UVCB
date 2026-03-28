# Building UVCB: a self-hosted voice, video, and text chat bridge in Go

**A hybrid MCU/transcoding pipeline — not a pure SFU — is the only viable architecture for UVCB.** The heterogeneous codec landscape (Opus on Discord/Mumble/Matrix, AAC on RTMP, SILK on QQ) and the hard requirement for composite RTMP output to VRChat make pure forwarding architectures structurally impossible. The critical insight from studying every major real-time media project (LiveKit, mediasoup, Janus, mumble-discord-bridge) is that even the most sophisticated Go WebRTC systems use **external transcoding engines** — FFmpeg or GStreamer — rather than attempting codec conversion in pure Go. UVCB should follow this proven pattern: a pure-Go core handling protocol connections and Opus audio mixing, with FFmpeg as a subprocess for RTMP muxing and the small number of transcoding operations actually required.

The good news is that **four of six voice platforms share Opus at 48 kHz natively**, meaning Discord ↔ Mumble ↔ Matrix ↔ XMPP audio can be relayed with zero transcoding. Only the RTMP/VRChat boundary (requiring AAC) and QQ (requiring SILK) need dedicated transcoding — a dramatically smaller CPU budget than a naive "transcode everything" design would suggest.

---

## Why a pure SFU fails and a selective MCU succeeds

An SFU forwards RTP packets without decoding — brilliant for homogeneous WebRTC conferences, but structurally incompatible with UVCB's requirements. VRChat needs a **single composite RTMP stream** containing mixed audio from all participants and a video track showing the chat overlay. An SFU cannot produce this; it only forwards individual tracks. Mediasoup's documentation states explicitly that it "doesn't have the ability to perform transcoding," and LiveKit handles RTMP output through a completely separate Egress service built on GStreamer.

A full MCU (decode everything → mix → re-encode) solves the codec problem universally but burns CPU proportional to every participant on every output. For self-hosters running on a 2-core VPS, this is prohibitive when bridging video.

The **selective MCU** (or "bridge-per-platform") architecture occupies the sweet spot. Each platform connector handles its own protocol, feeding decoded audio into a central PCM mixer. The mixer produces per-output streams: **Opus packets forwarded directly** between Discord/Mumble/Matrix/XMPP (zero transcoding), and **PCM → AAC encoding only for the RTMP output**. Video transcoding happens only when a VP8/VP9 source must reach RTMP — and this can be entirely avoided by negotiating H.264 as the preferred codec on all WebRTC endpoints.

Janus Gateway's architecture validates this approach. Its `audiobridge` plugin mixes Opus audio server-side while its `streaming` plugin handles external output — a modular hybrid that maps directly to UVCB's needs. The mumble-discord-bridge project, with **~190 GitHub stars**, demonstrates the simplest form: since both Mumble and Discord speak Opus at 48 kHz, it passes packets with minimal processing, using a jitter buffer at the protocol boundary and separate goroutines per direction.

---

## The Go media library landscape: what exists, what works, what doesn't

### Pion WebRTC is the foundation — but only the foundation

**Pion** (`github.com/pion/webrtc`, **~16,100 stars**, v4.2.9 as of February 2026, MIT license) is the industry-standard pure-Go WebRTC stack. It handles ICE/STUN/TURN connectivity, DTLS, SRTP, SDP negotiation, and RTP packet routing — all without CGo. Every Pion sub-package (`pion/rtp`, `pion/rtcp`, `pion/sdp`, `pion/ice`, `pion/dtls`, `pion/srtp`) is pure Go, enabling trivial cross-compilation to any GOOS/GOARCH target.

What Pion **cannot** do is equally important: no audio mixing, no codec transcoding, no Opus encoding (decode only via `pion/opus`), no RTMP support, and no video compositing. Pion's maintainer Sean Dubois positions it explicitly as ideal for "protocol bridging" — providing the transport layer while leaving media processing to purpose-built tools. The Pion `twitch` example and `Sean-Der/rtmp-to-webrtc` project demonstrate this pattern: Pion handles WebRTC, FFmpeg handles RTMP muxing.

### Opus codec libraries: three tiers of trade-offs

| Library | Encode | Decode | Pure Go? | Performance | Best for |
|---------|--------|--------|----------|-------------|----------|
| `pion/opus` | ❌ | ✅ | ✅ Yes | Good | Decode-only paths, zero CGo builds |
| `hraban/opus` (CGo) | ✅ | ✅ | ❌ Requires libopus | Native speed | Production encode/decode |
| `hraban/opus` (WASM/wazero) | ✅ | ✅ | ✅ via WASM | ~70-80% native | Cross-compilable encode/decode |

The **WASM variant** of `hraban/opus` is the most interesting development. By compiling libopus to WebAssembly and running it through wazero (a pure-Go WASM runtime), it achieves full Opus encode/decode at **70-80% of native speed with zero CGo dependency**. For UVCB — which needs to cross-compile to ARM, Windows, and Linux — this eliminates the single largest CGo pain point. The `github.com/godeps/opus` package implements this approach and is goroutine-safe.

The `pion/opus` pure-Go decoder handles the most common path (receiving Opus from WebRTC peers) without any native dependencies. For the re-encoding path (PCM → Opus for output to Mumble/Discord), either the WASM variant or CGo `hraban/opus` is required.

### RTMP: yutopp/go-rtmp wins on library quality

**`github.com/yutopp/go-rtmp`** (~397 stars, v0.0.6, pure Go) is the best library-grade RTMP implementation. It provides clean server and client APIs with a handler pattern, handles the full RTMP handshake/chunk protocol, and is used in production by **LiveKit's ingress service**. For a standalone streaming server, `github.com/q191201771/lal` (~2,900 stars, pure Go, actively maintained) offers RTMP + RTSP + HLS in one package. `gwuhaolin/livego` (~9,900 stars) is popular but designed as a standalone application, not an embeddable library.

### FFmpeg integration: subprocess beats CGo for this use case

**`github.com/u2takey/ffmpeg-go`** (~2,300 stars) wraps FFmpeg as a subprocess with a fluent Go API, but its maintenance has stalled (no releases in 12+ months). For UVCB, using `os/exec` directly is more reliable and transparent. The critical FFmpeg pipelines are straightforward:

**Mixed audio → AAC → RTMP:**
```bash
ffmpeg -f s16le -ar 48000 -ac 2 -i pipe:0 -c:a aac -b:a 128k -f flv rtmp://server/live/key
```

**Chat overlay frames + mixed audio → H.264+AAC → RTMP:**
```bash
ffmpeg -f rawvideo -pix_fmt rgba -s 1280x720 -r 30 -i pipe:0 \
  -f s16le -ar 48000 -ac 2 -i pipe:3 \
  -c:v libx264 -preset ultrafast -tune zerolatency -pix_fmt yuv420p \
  -c:a aac -b:a 128k -f flv rtmp://server/live/key
```

**CGo FFmpeg bindings** (`github.com/asticode/go-astiav`, ~400 stars, actively maintained, compatible with FFmpeg n8.0) offer lower latency through zero-copy frame handling but massively complicate the build process. For UVCB's self-hosted audience, subprocess FFmpeg is the right choice — users can install FFmpeg through their package manager, and it only needs to run when RTMP output is active.

### Chat overlay rendering: Go canvas beats headless Chrome

For rendering the chat UI as a video stream, **`github.com/fogleman/gg`** (~4,400 stars, pure Go) is the pragmatic choice. It renders text, shapes, and images to `image.RGBA` buffers that pipe directly to FFmpeg. A chat overlay is fundamentally simple — usernames, timestamps, message text with colored backgrounds — and doesn't need CSS layout or web fonts.

**Headless Chromium** (via `github.com/chromedp/chromedp`, ~11,000 stars, pure Go control layer) is what LiveKit and Mux use for composite recording, but it requires ~200-500 MB RAM per Chrome instance and adds significant latency. For UVCB's self-hosted resource constraints, the Go canvas approach uses **~10 MB overhead** and produces deterministic frame timing. Reserve headless Chrome as an optional "rich overlay" mode for users with spare resources.

---

## The codec compatibility matrix reveals a surprisingly efficient architecture

### Audio: Opus is the lingua franca

| Source → Dest | Discord | Mumble | Matrix | XMPP (modern) | RTMP/VRChat | QQ |
|---|---|---|---|---|---|---|
| **Discord** (Opus 48k stereo) | ✅ | ✅* | ✅ | ✅** | ❌ Opus→AAC | ❌ Opus→SILK |
| **Mumble** (Opus 48k mono) | ✅* | ✅ | ✅* | ✅** | ❌ | ❌ |
| **Matrix** (Opus 48k stereo) | ✅ | ✅* | ✅ | ✅** | ❌ | ❌ |
| **RTMP** (AAC 44.1/48k) | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| **QQ** (SILK 24k) | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |

✅ = direct passthrough, ✅* = channel count adjustment only (mono↔stereo, trivial), ✅** = Opus negotiable but may fall back to G.711, ❌ = full transcode required.

**The key finding**: Discord, Mumble, Matrix, and modern XMPP clients all speak **Opus at 48 kHz**. Between these four platforms, UVCB can forward raw Opus packets with zero transcoding — only adjusting mono/stereo channel count where needed (a trivial PCM operation). This is exactly how mumble-discord-bridge achieves its efficiency.

Only two boundaries require actual transcoding: **Opus → AAC** at the RTMP/VRChat output (CPU cost: ~5-8% of one core per stream), and **Opus → SILK** for QQ (similar cost plus a 48→24 kHz resample). No Go library exists for SILK encoding — this would require CGo wrapping of the Silk SDK or an FFmpeg subprocess.

### Video: negotiate H.264 everywhere to avoid transcoding entirely

For video, **H.264 is the universal codec**. Discord, Matrix, and XMPP all support it via WebRTC negotiation. RTMP requires it. If UVCB forces H.264 as the preferred video codec in all WebRTC SDP offers, video frames can pass through to RTMP without any transcoding — a strategy that reduces per-stream CPU cost from "saturate a core at 720p" to "near zero."

VP8→H.264 software transcoding costs approximately **100-300% of a single core for 720p30**. VP9→H.264 is even worse. Hardware encoding (NVENC, VAAPI, QSV) reduces this to ~10-20% of a core but requires specific GPU hardware that most self-hosters won't have. The smart architectural choice is to avoid the problem entirely through codec negotiation.

---

## VRChat integration: ambitious but achievable with clear constraints

### The video player path works, with caveats

VRChat's **AVPro Video Player** (the only option for live streams) supports RTMP, RTSP, HLS, and MPEG-TS. On PC, **RTSP delivers the lowest latency at ~1-3 seconds** — Japanese VRChat communities using TopazChat have achieved sub-1-second with optimized settings. HLS adds 5-30 seconds of latency. Quest/Android requires HTTPS for video playback, making MPEG-TS or HLS the only cross-platform options.

The critical constraint is VRChat's **URL allowlist system**. Self-hosted stream URLs are "untrusted" by default. Users must either enable "Allow Untrusted URLs" in their VRChat settings, or the world creator must add the bridge server domain to the world's allowlist (maximum **10 domains** per world). In public instances since December 2024, untrusted URLs are blocked entirely unless world-allowlisted. For a self-hosted tool, the practical path is to document the allowlist requirement and optionally support VRCDN (a paid relay service that's pre-allowlisted).

The stream can carry **both video (chat overlay) and audio (voice bridge) simultaneously** as a standard RTMP/RTSP stream. AVPro outputs audio in-world, so voice bridge participants are audible through the video player's spatial audio. The recommended format is **H.264 Baseline/Main profile at 720p, AAC stereo at 128 kbps, 2500-4000 kbps total bitrate, 1-2 second keyframe interval**.

For the RTMP relay server, **SRS** (Simple Realtime Server, MIT licensed, actively maintained at v7.0) is the strongest choice. It outputs RTSP (for PC, lowest latency), MPEG-TS (for Quest), and HLS (fallback) from a single RTMP ingest. Docker deployment is one command: `docker run -p 1935:1935 -p 8080:8080 ossrs/srs:6`.

### Udon scripting has hard limitations that shape the world design

Udon's sandbox restrictions are the tightest constraint on the in-world experience. **VRCUrl objects cannot be constructed dynamically at runtime** — stream URLs must be predefined at build time. There is no HTTP POST, no WebSocket, no server push. The only outbound data path is **VRCStringDownloader** (text GET requests) and **VRCImageDownloader**, both sharing a **5-second global cooldown**. This means the bridge world must use a fixed stream endpoint URL and can poll the bridge API for status updates at most every 5 seconds.

The practical architecture for a bridge world: pre-configure the RTMP stream URL at build time, use VRCStringDownloader to poll a JSON status endpoint (who's connected, chat messages) every 5-10 seconds, and optionally use VRCImageDownloader to fetch a server-rendered chat overlay image as a texture.

### The overlay fallback for any world

For users not in the custom bridge world, **OVR Toolkit** ($12.99, Steam, Very Positive with 3,053 reviews) is the best option. Its **Custom Apps** feature allows loading web applications directly as VR overlays — the Nuxt dashboard would display as a floating, interactive panel in any VR world. **Desktop+** (free, open source) offers built-in browser overlays for users who prefer a free option. **XSOverlay** ($9.99, most popular) requires capturing a desktop browser window rather than loading URLs directly.

None of these overlay tools relay audio bidirectionally — they are visual only. Voice bridge audio must come through either the in-world video player or OS-level audio routing (virtual audio cable piping bridge output to VRChat's microphone input).

**VRChat's OSC chatbox** (`/chatbox/input` on UDP port 9000) provides a supplementary text path: 144 characters maximum, ASCII only, ~1.5-second update rate, visible above the user's avatar. It's useful for displaying "Bridge connected: 5 users" but not for full chat relay.

---

## Lessons from existing projects

### mumble-discord-bridge proves the Opus passthrough pattern

This Go project (~190 stars, using `disgoorg/disgo` for Discord and `stieneee/gumble` for Mumble) demonstrates that when codecs match, **simple packet relay with a jitter buffer is all you need**. It runs separate goroutines per direction (`DiscordDuplex`, `MumbleDuplex`) communicating through Go channels — a pattern UVCB should replicate for each platform connector. Its key limitation is audio-only, single-channel-pair bridging, and a requirement for CGo Opus bindings (`stieneee/gopus`).

### LiveKit proves FFmpeg/GStreamer is mandatory for RTMP

LiveKit (~22,000 stars, Apache 2.0, extremely active) is the most sophisticated Go WebRTC project. Its core SFU is pure Go built on Pion, but its Egress service (RTMP output, recording) uses **GStreamer** — chosen over FFmpeg for "greater flexibility, programmatic control via go-gst, and most importantly, robust error handling." The Egress service's Room Composite mode renders in headless Chrome, captures via XVFB, and encodes to RTMP through GStreamer. This confirms that no pure-Go path exists for production RTMP output.

LiveKit's **server-sdk-go** (`github.com/livekit/server-sdk-go/v2`) is embeddable as a Go library and could theoretically allow UVCB to connect to a LiveKit room as a bot participant, but this adds a heavyweight dependency. UVCB is better served using Pion directly.

### mediasoup confirms that SFUs punt on transcoding

mediasoup (Node.js + C++ workers) explicitly states it "doesn't have the ability to perform transcoding" and relies on FFmpeg/GStreamer connected via PlainTransport for any codec conversion. Its Producer/Consumer model with Routers per media group is architecturally clean and worth studying, but the core lesson is the same: media processing lives outside the SFU.

---

## Self-hosted deployment: one binary, one config file, FFmpeg optional

### Hardware requirements by deployment tier

| Tier | Specs | Capability |
|------|-------|------------|
| **Raspberry Pi 4** | 4 cores ARM, 4 GB | Audio-only bridging, ~10-20 voice streams, no video |
| **Budget VPS** | 2 cores, 2 GB RAM, 1 Gbps | Audio bridge + basic RTMP output (chat overlay + mixed voice) |
| **Recommended** | 4 cores, 8 GB RAM, 1 Gbps | Full audio + video bridge, RTMP output, ~20-30 participants |
| **Heavy use** | 8 cores, 16 GB RAM, 10 Gbps | Multiple rooms, video transcoding, recording |

**Bandwidth math**: Opus voice at 64 kbps plus overhead is ~100 kbps per participant per direction. Ten voice users require ~2 Mbps aggregate. RTMP output at 720p adds 3-6 Mbps. Video bridging at 360p is ~500 kbps per stream, making 10 video users ~10 Mbps. Almost any modern VPS handles voice-only; video demands real bandwidth.

### Distribution follows the Mumble/Caddy model

UVCB should ship as a **single Go binary** with an optional Docker Compose file. Go's cross-compilation (`GOOS=linux GOARCH=arm64 go build`) produces static binaries with zero runtime dependencies for the core audio bridge. FFmpeg becomes a required dependency only when RTMP output or video transcoding is enabled — and it can be bundled in the Docker image while remaining optional for bare-metal installs.

**Embed `pion/turn`** (~2,200 stars, pure Go, v5.0.3, explicitly designed for embedding) directly in the binary to eliminate the need for separate coturn deployment. This is a major UX win: ~15-25% of WebRTC connections require TURN relay, and expecting self-hosters to deploy and configure coturn separately is a significant adoption barrier. The embedded TURN server should support TURN/TLS on port 443 for maximum firewall traversal.

For TLS, recommend **Caddy** as the default reverse proxy. LiveKit uses this exact approach for self-hosted deployments. A three-line Caddyfile provides automatic Let's Encrypt certificates with zero manual configuration. For development, support a `--insecure` flag. For private networks, Tailscale's automatic TLS certs for `*.ts.net` domains provide an elegant path.

Configuration should use **YAML with environment variable overrides**, following LiveKit's pattern. The minimal config is a domain name and an API secret — everything else has sensible defaults. Audio codec, port numbers, TURN configuration, and platform connectors should auto-configure where possible and expose tuning knobs only for advanced users.

---

## Concrete architecture recommendation with specific libraries

### The recommended component stack

| Component | Library | Stars | Pure Go? | Role |
|-----------|---------|-------|----------|------|
| WebRTC transport | `pion/webrtc` v4 | 16.1k | ✅ | Matrix/Element Call, XMPP Jingle connections |
| RTP/RTCP handling | `pion/rtp`, `pion/rtcp` | (part of Pion) | ✅ | Packet parsing, payloading/depayloading |
| Opus decode | `pion/opus` | (part of Pion) | ✅ | Decode incoming Opus → PCM (no CGo) |
| Opus encode/decode | `hraban/opus` WASM variant | ~400 | ✅ via wazero | Full encode/decode without CGo |
| RTMP protocol | `yutopp/go-rtmp` | 397 | ✅ | RTMP server/client for VRChat pipeline |
| Discord API | `disgoorg/disgo` | ~500 | ✅ | Discord bot voice connection |
| Mumble client | `stieneee/gumble` | (fork) | ✅ (core) | Mumble protocol client |
| Chat overlay | `fogleman/gg` | 4.4k | ✅ | Render chat UI as image frames |
| TURN server | `pion/turn` v5 | 2.2k | ✅ | Embedded NAT traversal |
| Transcoding/RTMP mux | FFmpeg (subprocess) | N/A | N/A | Opus→AAC, frame encoding, RTMP output |
| RTMP relay server | SRS v7 (sidecar) | ~25k | N/A | RTSP/HLS/MPEG-TS delivery to VRChat |

### The data flow

The architecture has three distinct pipelines running as goroutines:

**Audio pipeline** (always active): Each platform connector decodes incoming audio to PCM 48 kHz (or forwards raw Opus packets when source/destination codecs match). A central mixer sums PCM samples from all active sources with clipping protection. Per-output encoders produce: Opus packets for Discord/Mumble/Matrix/XMPP (direct passthrough where possible), and PCM piped to FFmpeg for AAC encoding when RTMP output is active. Audio mixing of N int16 PCM streams is trivially implementable in Go — no library needed, just additive mixing with clamping to `[-32768, 32767]`.

**Video/overlay pipeline** (active when RTMP output is enabled): `fogleman/gg` renders the chat overlay as RGBA frames at 30 fps. Frames pipe to FFmpeg's stdin as raw video. FFmpeg encodes to H.264 (ultrafast preset, zerolatency tune), muxes with the AAC audio from the audio pipeline, and pushes the combined FLV stream to the RTMP relay server. If a participant's webcam/screen video is being bridged, H.264 frames pass through without re-encoding (assuming H.264 was negotiated via SDP).

**Text pipeline** (always active, lightest weight): Messages from all platforms flow through a central message bus. Each platform connector publishes and subscribes to messages. IRC and text-only channels feed directly into the chat overlay renderer and the Nuxt dashboard's WebSocket feed.

### The phased implementation path

**Phase 1 — Audio-only bridge (minimum viable product):** Discord ↔ Mumble voice bridge using Opus passthrough, following mumble-discord-bridge's architecture. Add Matrix via Pion WebRTC. Text bridging across all platforms. Pure Go binary, no FFmpeg dependency. This handles the most common community use case.

**Phase 2 — RTMP output for VRChat:** Add FFmpeg subprocess for PCM → AAC encoding and RTMP muxing. Add `fogleman/gg` chat overlay rendering. Add SRS as a Docker sidecar for stream delivery. This enables the VRChat world integration.

**Phase 3 — Video bridging:** Add WebRTC video track handling via Pion. Negotiate H.264 everywhere for passthrough. Compose multi-participant video layout (selected speaker or grid) using Go image compositing or FFmpeg's filter_complex. This is the most CPU-intensive feature and should be optional.

**Phase 4 — Advanced features:** XMPP Jingle voice, QQ SILK transcoding (CGo), hardware-accelerated encoding (NVENC/VAAPI via FFmpeg flags), HLS output for Quest compatibility, adaptive quality degradation based on CPU load.

---

## What remains genuinely hard

Three challenges have no clean library solution today. **QQ voice integration** requires SILK codec support — no Go library exists, the protocol is undocumented, and Tencent's modified SILK format uses proprietary headers. This will likely require reverse-engineering and CGo wrapping. **Dynamic VRChat video player URLs** cannot be constructed at runtime in Udon — the stream endpoint must be hardcoded at world build time, meaning each bridge deployment needs its own world upload. **Full-duplex VRChat voice** (bridge audio out through the video player, bridge audio in from VRChat's microphone) requires users to configure OS-level audio routing (virtual audio cables), as VRChat provides no programmatic microphone access.

The architecture above is designed to be **incrementally buildable**: Phase 1 produces a useful audio bridge with zero transcoding overhead and no FFmpeg dependency, while each subsequent phase adds capabilities that justify their CPU and complexity costs. For a self-hosted open-source tool, this progressive complexity curve is essential — the user running a Raspberry Pi for their friend group's Mumble-Discord bridge should never need to configure GStreamer pipelines.
