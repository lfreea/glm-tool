# GLM Tool

[English](README.md)

OpenAI API 兼容的智谱清言代理中间件。

## 功能特性

- **API 兼容**：完全兼容 OpenAI API 格式，可无缝对接现有工具
- **请求透传**：灵活转发，支持任意 JSON 字段
- **流式响应**：支持 SSE 流式输出
- **图片识别**：自动识别图片并转换为文本描述
- **智能缓存**：相同图片只识别一次，24小时持久化缓存
- **Anthropic 兼容**：同时支持 Anthropic Messages API 格式

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/v1/models` | GET | 获取模型列表 |
| `/v1/chat/completions` | POST | 聊天补全（OpenAI 格式） |
| `/v1/messages` | POST | 消息接口（Anthropic 格式） |
| `/v1/messages/count_tokens` | POST | Token 计数 |

## 部署

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `PORT` | 服务端口 | 8080 |
| `TARGET_API_URL` | 目标 API 地址 | https://open.bigmodel.cn/api/coding/paas/v4 |
| `LOG_LEVEL` | 日志级别 | info |
| `DEBUG` | Debug 模式 | false |
| `CACHE_PATH` | 缓存文件路径 | image_cache.db |
| `CACHE_TTL_HOURS` | 缓存保留时间（小时） | 24 |

### 方式一：二进制部署

1. 从 [Releases](https://github.com/lfreea/glm-tool/releases) 下载对应平台的二进制文件
2. 复制配置文件并修改

```
cp .env.example .env
```

3. 运行服务

```
./glm-tool-linux-amd64
```

### 方式二：Docker 部署

```
docker run -d \
  --name glm-tool \
  -p 8080:8080 \
  -e PORT=8080 \
  -e TARGET_API_URL=https://open.bigmodel.cn/api/coding/paas/v4 \
  ghcr.io/lfreea/glm-tool:latest
```

### 方式三：Docker Compose

创建 `docker-compose.yml`：

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

启动服务：

```
docker compose up -d
```

### 方式四：从源码编译

1. 克隆仓库

```
git clone https://github.com/lfreea/glm-tool.git
cd glm-tool
```

2. 编译

```
go build -o glm-tool ./cmd/server
```

3. 运行

```
./glm-tool
```

## 使用说明

### 认证方式

每次请求需要在 `Authorization` header 中携带智谱清言的 API Key：

```
Authorization: Bearer your_api_key_here
```

### 图片识别

默认启用，无需配置。当请求中包含图片时，自动识别并转换为文本描述。

**特性：**
- 自动检测 `image_url` 类型
- 智能缓存，相同图片只识别一次
- 缓存 24 小时有效，重启后仍可用

## 许可证

[MIT](LICENSE)
