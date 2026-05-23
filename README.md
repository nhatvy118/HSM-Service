# HSM Server

HTTP API wrap SoftHSM2 (PKCS#11), expose endpoint `generate / sign / verify / export public key` cho BE đối tác gọi qua HTTPS — partner **không cần biết PKCS#11**, không có PIN, không có `.so`.

Dev với SoftHSM (giả lập), production có thể migrate sang HSM thật (AWS CloudHSM, Thales Luna...) **không phải sửa code Go** — chỉ đổi env `PKCS11_MODULE_PATH` + cài vendor client.

---

## Kiến trúc

```
┌─ Partner BE ─────────┐
│  HTTP client          │
└─────────┬─────────────┘
          │ HTTPS + X-API-Key
          ▼
┌─ hsm-server container ────────────────────────────┐
│  Go HTTP server :8080                              │
│  ├── handler/    (HTTP layer, JSON in/out)         │
│  ├── usecase/    (HSM business logic, crypto11)    │
│  └── domain/     (DTO)                             │
│       ↓ PKCS#11 (load libsofthsm2.so)              │
└────────┬───────────────────────────────────────────┘
         │ file I/O (cùng host)
         ▼
┌─ softhsm container ─────────────────────────────┐
│  init token (1 lần), giữ token files            │
└─────────────────────────────────────────────────┘
         ▲
         │ shared volume "softhsm-tokens"
         │ (cả 2 container đọc/ghi cùng folder)
```

- **softhsm container**: init token + giữ key files trong volume
- **hsm-server container**: Go HTTP server, load `libsofthsm2.so` riêng, đọc cùng token files

---

## Prerequisites

- Docker Desktop / Docker + Docker Compose v2
- `curl` + `jq` cho test (`brew install jq`)
- (Optional) `openssl` để verify signature ngoài HSM

---

## Quick start

### 1. Cấu hình env

```bash
cp .env.example .env
```

Nội dung `.env`:
```bash
TOKEN_LABEL=dev-ecdsa
USER_PIN=1234
SO_PIN=12345678
API_KEY=change-me-in-prod
```

⚠️ Đây là **dev PIN**. Production lấy từ Vault / Secrets Manager, không hardcode.

### 2. Build + start

```bash
docker compose up -d --build
```

Đợi ~30s. Verify:
```bash
docker compose ps
# softhsm      Up (healthy)
# hsm-server   Up
```

## API endpoints

Base URL: `http://localhost:8080`
Auth: header `X-API-Key: <key>` (trừ `/health`)

| Method | Path | Body | Response |
|--------|------|------|----------|
| `GET` | `/health` | — | `{"status":"ok"}` |
| `POST` | `/v1/keys` | `{label, type}` | `{label, id, type}` |
| `GET` | `/v1/keys/{label}/pubkey` | — | `{pem}` (PEM SubjectPublicKeyInfo) |
| `POST` | `/v1/keys/{label}/sign` | `{data, algorithm}` | `{signature}` (base64 DER) |
| `POST` | `/v1/keys/{label}/verify` | `{data, signature, algorithm}` | `{valid}` |

**Key types supported:** `ECDSA-P256`, `ECDSA-P384`, `RSA-2048`, `RSA-4096`

**Algorithms supported:** `ECDSA-SHA256` (v1)

### Curl mẫu

```bash
# Generate
curl -X POST http://localhost:8080/v1/keys \
  -H "X-API-Key: change-me-in-prod" \
  -H "Content-Type: application/json" \
  -d '{"label":"my-key","type":"ECDSA-P256"}'

# Sign ("hello hsm" → base64)
curl -X POST http://localhost:8080/v1/keys/my-key/sign \
  -H "X-API-Key: change-me-in-prod" \
  -H "Content-Type: application/json" \
  -d '{"data":"aGVsbG8gaHNt","algorithm":"ECDSA-SHA256"}'

# Verify (paste signature từ bước trên)
curl -X POST http://localhost:8080/v1/keys/my-key/verify \
  -H "X-API-Key: change-me-in-prod" \
  -H "Content-Type: application/json" \
  -d '{"data":"aGVsbG8gaHNt","signature":"MEUCIQ...","algorithm":"ECDSA-SHA256"}'

# Export public key (để verify ngoài HSM)
curl http://localhost:8080/v1/keys/my-key/pubkey \
  -H "X-API-Key: change-me-in-prod"
```

### Error responses

| Status | Khi nào |
|--------|---------|
| 400 | Body sai, base64 hỏng, algorithm/key-type không support |
| 401 | Sai/thiếu `X-API-Key` |
| 404 | Key label không tồn tại |
| 409 | Tạo key với label đã có |
| 500 | Lỗi HSM hoặc internal |

---

## Cấu trúc project

```
HSM/
├── deployment/
│   ├── softhsm/             # softhsm container (init token + giữ tokens)
│   │   ├── Dockerfile
│   │   ├── softhsm2.conf
│   │   └── entrypoint.sh
│   └── server/
│       └── Dockerfile       # hsm-server container (Go + softhsm2 lib)
│
├── internal/                # Go application code
│   ├── main.go              # bootstrap (load config, init HSM, start server)
│   ├── config.go            # env loading
│   ├── middleware.go        # auth (X-API-Key), request log
│   ├── go.mod, go.sum
│   ├── domain/entity/
│   │   └── entity.go        # DTO (request/response JSON shapes)
│   ├── handler/
│   │   └── handler.go       # HTTP handlers (parse → call usecase → response)
│   └── usecase/
│       └── usecase.go       # HSM business logic (crypto11 wrapper)
│
├── docker-compose.yml       # softhsm + hsm-server services
├── .env.example             # template env vars
├── test-api.sh              # smoke test (generate → sign → verify)
└── README.md
```

---

## Configuration (env vars)

### softhsm container

| Var | Mục đích |
|-----|----------|
| `TOKEN_LABEL` | Tên token sẽ init (vd: `dev-ecdsa`) |
| `USER_PIN` | PIN cho operator role (sign/encrypt) |
| `SO_PIN` | PIN cho Security Officer (init/reset token) |

### hsm-server container

| Var | Mục đích |
|-----|----------|
| `PKCS11_MODULE_PATH` | Path tới `.so` (default: `/usr/lib/softhsm/libsofthsm2.so`) |
| `HSM_TOKEN` | Token label cần login (= `TOKEN_LABEL`) |
| `HSM_USER_PIN` | PIN để login token (= `USER_PIN`) |
| `API_KEY` | Header `X-API-Key` partner phải gửi |
| `PORT` | Port HTTP server (default: `8080`) |

---

## Development workflow

### Sửa code Go

```bash
# Sửa file trong internal/
vim internal/usecase/usecase.go

# Rebuild + run lại hsm-server (không động softhsm)
docker compose up -d --build hsm-server

### Reset state (xóa hết token, key)

```bash
docker compose down -v   # xóa volume tokens
docker compose up -d --build
```

### Xem logs realtime

```bash
docker compose logs -f hsm-server
```

### Inspect HSM trực tiếp

```bash
# List slots / tokens
docker exec softhsm softhsm2-util --show-slots

# List objects trong token
docker exec softhsm pkcs11-tool \
  --module /usr/lib/softhsm/libsofthsm2.so \
  --token-label dev-ecdsa --login --pin 1234 -O
```

---

## Security model

```
┌─ Layer 1: HTTP auth ─────────────────────┐
│  Partner ──X-API-Key──► hsm-server        │
│  Service revoke key 1 partner mà không    │
│  ảnh hưởng partner khác                   │
└──────────────────────────────────────────┘

┌─ Layer 2: HSM login ─────────────────────┐
│  hsm-server ──USER_PIN──► HSM             │
│  PIN không bao giờ rời service container  │
│  Partner KHÔNG biết PIN                   │
└──────────────────────────────────────────┘

┌─ Layer 3: Key protection ────────────────┐
│  Private key never leaves HSM             │
│  CKA_EXTRACTABLE = false                  │
│  Sign operation runs IN-HSM               │
└──────────────────────────────────────────┘
```

**Pattern**: generate-in-HSM (không có endpoint import). Key sinh trong HSM → không tồn tại bên ngoài → không thể bị copy.

---

## Limitations v1

| | v1 | Roadmap |
|---|----|---------| 
| Algorithm sign | ECDSA-SHA256 | Thêm RSA-PKCS1-SHA256/384/512 |
| Encrypt/Decrypt | ❌ | RSA-OAEP, AES-GCM |
| Multi-tenant | ❌ (1 token) | Multi-token, API key → token routing |
| Per-key ACL | ❌ | API key whitelist key labels |
| Import key | ❌ | (bỏ chủ ý — generate-in-HSM only) |
| Rate limit | ❌ | Middleware |
| Metrics | ❌ | Prometheus `/metrics` |


## License

Internal use only.
