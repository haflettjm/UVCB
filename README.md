# UVCB - Universal Virtual Chat Bridge

UVCB (Universal Virtual Chat Bridge) is a universal chat app bridge for connecting conversations across a wide range of messaging platforms—both popular and niche. It aims to unify communications across systems like Discord, IRC, Matrix, Mumble, and more into a single, streamlined backend written in Golang.

---

## 🎯 MVP Goals

### ✅ Core Chat Service Integrations (Phase 1)
These integrations are prioritized for the MVP release:

- [ ] **Mumble** - Support for low-latency voice relays and messaging via Murmur API.
- [ ] **IRC** - Basic text relay and bot functionality using IRC protocol.
- [ ] **VRChat** - Bridge via Open Sound Control (OSC) messaging or compatible interfaces.
- [ ] **Discord** - Two-way message sync via official bot API.
- [ ] **Matrix** - Federated room bridging and message conversion.
- [ ] **QQ** - Integration via third-party wrappers like `go-cqhttp`.
- [ ] **XMPP** - Legacy chat protocol bridging using standard XMPP libraries.

### ❓ Maybe (Phase 2+)
These are considered for future versions based on demand and feasibility:

- [ ] **Telegram** - Bot API and/or userbot bridge.
- [ ] **Slack, Signal, WhatsApp** - TBD based on demand.

---

## 🛠 Tech Stack

| Layer            | Technology        |
|------------------|------------------|
| Backend          | Golang            |
| Deployment       | Terraform (with TypeScript automation) |
| Frontend         | Nuxt + Vue.js     |

---

## 🚀 Deployment Targets
- Kubernetes (Preferred)
- Docker Compose
- Bare metal or cloud VM support (via Ansible/Terraform)

---

## 📂 Project Structure (Planned)
```
uvcb/
├── cmd/                  # Entry points for different services
├── internal/             # Core logic & adapters
│   ├── bridge/           # Message bridge implementations per platform
│   ├── core/             # Message queue, normalization, routing
│   └── config/           # Config loading & schema
├── web/                  # Nuxt frontend (status/dashboard)
├── deployments/          # Terraform & deployment manifests
│   ├── k8s/
│   └── compose/
├── scripts/              # CI/CD & helper scripts (TS)
└── README.md
```

---

## 📌 Implementation Steps

### Phase 1: Core Infrastructure
- [ ] Initialize Monorepo Structure
  - [ ] `go mod init uvcb`
  - [ ] Set up Nuxt app in `/web`
- [ ] Define Message Format Standard
  - [ ] Common normalized message object (sender, text, timestamp, metadata)
- [ ] Implement Core Router
  - [ ] Route messages between services using channels or a pubsub interface

### Phase 2: Backend Bridges (Golang)
- [ ] Implement Bridges
  - [ ] Discord (via bot API)
  - [ ] IRC (via `goirc` or similar)
  - [ ] Matrix (via `gomatrix`)
  - [ ] Mumble (via gRPC or Murmur ICE API)
  - [ ] XMPP (via `gosrc/xmpp`)
  - [ ] QQ (via a wrapper or bridge like `go-cqhttp`)
  - [ ] VRChat (OSC or overlay API)
- [ ] Bridge Lifecycle Management
  - [ ] Auto-reconnect, health checks, error queues

### Phase 3: Frontend + Dashboard
- [ ] Setup Nuxt + Vue UI
  - [ ] Display online status, message logs, and bridge statuses
  - [ ] Manual message relay tool (for admin use)

### Phase 4: Deployment Pipeline
- [ ] Create Terraform Scripts
  - [ ] Kubernetes manifests for all components
  - [ ] TS-driven deploy flows
- [ ] CI/CD
  - [ ] GitHub Actions for automated build/test/deploy

### Phase 5: Monitoring & Logs
- [ ] Integrate Logging (Zap/Slog)
  - [ ] Add Prometheus metrics for bridge health
  - [ ] Grafana dashboards

---

## 📜 License
MIT License (or similar open-source license to be chosen)

---

## 🤝 Contributing
- Contributions are welcome.
- Each bridge should live in its own package and implement a shared interface.

---

## 📫 Contact
Project by Edji-Ideas LLC. Maintained by Jacob Haflett.

Join us as we unify the fragmented world of messaging!
