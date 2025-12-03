# GLM Tool

[中文文档](README_CN.md)

OpenAI API compatible proxy middleware for Zhipu AI (GLM).

## Features

- **API Compatible**: Fully compatible with OpenAI API format, seamless integration with existing tools
- **Request Passthrough**: Flexible forwarding, supports any JSON fields
- **Streaming Response**: SSE streaming output support
- **Image Recognition**: Automatic image recognition and text conversion
- **Smart Caching**: Same images are recognized only once, 24-hour persistent cache
- **Anthropic Compatible**: Also supports Anthropic Messages API format

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/v1/models` | GET | List models |
| `/v1/chat/completions` | POST | Chat completions (OpenAI format) |
| `/v1/messages` | POST | Messages (Anthropic format) |
| `/v1/messages/count_tokens` | POST | Token counting |

## Deployment

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Service port | 8080 |
| `TARGET_API_URL` | Target API URL | https://open.bigmodel.cn/api/coding/paas/v4 |
| `LOG_LEVEL` | Log level | info |
| `DEBUG` | Debug mode | false |
| `CACHE_PATH` | Cache file path | image_cache.db |
| `CACHE_TTL_HOURS` | Cache retention time (hours) | 24 |

### Option 1: Binary Deployment

1. Download the binary for your platform from [Releases](https://github.com/lfreea/glm-tool/releases)
2. Copy and edit configuration

```
cp .env.example .env
```

3. Run the service

```
./glm-tool-linux-amd64
```

### Option 2: Docker

```
docker run -d \
  --name glm-tool \
  -p 8080:8080 \
  -e PORT=8080 \
  -e TARGET_API_URL=https://open.bigmodel.cn/api/coding/paas/v4 \
  ghcr.io/lfreea/glm-tool:latest
```

### Option 3: Docker Compose

Create `docker-compose.yml`:

```yaml
services:
  glm-tool:
    image: ghcr.io/lfreea/glm-tool:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - TARGET_API_URL=https://open.bigmodel.cn/api/coding/paas/v4
    restart: unless-stopped
```

Start the service:

```
docker compose up -d
```

### Option 4: Build from Source

1. Clone the repository

```
git clone https://github.com/lfreea/glm-tool.git
cd glm-tool
```

2. Build

```
go build -o glm-tool ./cmd/server
```

3. Run

```
./glm-tool
```

## Usage

### Authentication

Each request requires the Zhipu AI API Key in the `Authorization` header:

```
Authorization: Bearer your_api_key_here
```

### Image Recognition

Enabled by default, no configuration needed. When images are detected in requests, they are automatically recognized and converted to text descriptions.

**Features:**
- Auto-detects `image_url` type
- Smart caching, same images recognized only once
- Cache valid for 24 hours, persists after restart

## License

[MIT](LICENSE)
