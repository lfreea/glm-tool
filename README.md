# GLM Tool - OpenAI API 转发中间件

这是一个使用 Go + Gin 实现的工具中间件，用于接收 OpenAI API 格式的请求并透传转发到智谱清言的 API 端点。

## 功能特性

- ✅ 接收 OpenAI API 格式的请求
- ✅ 使用 `map[string]any` 透传请求数据（无结构体限制）
- ✅ 转发请求到 `https://open.bigmodel.cn/api/coding/paas/v4`
- ✅ 支持配置管理
- ✅ 健康检查端点
- ✅ **图片自动识别**：自动识别并转换图片为文本描述
- ✅ **智能缓存**：相同图片只识别一次，24小时持久化缓存

## 项目结构

```
glm-tool/
├── cmd/
│   └── server/          # 主程序入口
│       └── main.go
├── config/              # 配置管理
│   └── config.go
├── internal/
│   ├── handler/         # HTTP 处理器
│   │   └── handler.go
│   └── proxy/           # 请求转发
│       └── proxy.go
├── .env.example         # 配置文件示例
├── test.sh              # 测试脚本
├── go.mod
└── README.md
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置环境变量

复制 `.env.example` 到 `.env` 并填写配置：

```bash
cp .env.example .env
```

编辑 `.env` 文件：

```env
PORT=8080
TARGET_API_URL=https://open.bigmodel.cn/api/coding/paas/v4
LOG_LEVEL=info
```

**注意**：API Key 不再通过配置文件设置，而是在每次请求时通过 `Authorization` header 传递。

### 3. 运行服务

```bash
# 直接运行
go run cmd/server/main.go

# 或者编译后运行
go build -o bin/server cmd/server/main.go
./bin/server
```

服务将在 `http://localhost:8080` 启动。

### 4. 测试服务

```bash
# 编辑 test.sh，替换 API_KEY 为你的实际 API Key
vim test.sh

# 运行测试脚本
./test.sh

# 或者手动测试健康检查
curl http://localhost:8080/health
```

## API 端点

### 1. 健康检查

```bash
GET /health
```

响应示例：
```json
{
  "status": "ok",
  "service": "glm-tool"
}
```

### 2. Models 列表

```bash
GET /v1/models
Authorization: Bearer <your_api_key>
```

**重要**：每次请求必须在 `Authorization` header 中携带 API Key。

请求示例：
```bash
curl -X GET http://localhost:8080/v1/models \
  -H "Authorization: Bearer your_api_key_here"
```

响应格式：直接透传目标 API 的响应

响应示例：
```json
{
  "object": "list",
  "data": [
    {
      "id": "glm-4",
      "object": "model",
      "created": 1234567890,
      "owned_by": "zhipuai"
    }
  ]
}
```

### 3. Chat Completions

```bash
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer <your_api_key>
```

**重要**：每次请求必须在 `Authorization` header 中携带 API Key。

**支持流式响应**：设置 `"stream": true` 可以获得流式响应（SSE 格式）。

#### 非流式请求示例：

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_api_key_here" \
  -d '{
    "model": "glm-4",
    "messages": [
      {
        "role": "user",
        "content": "你好"
      }
    ],
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

#### 流式请求示例：

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_api_key_here" \
  -d '{
    "model": "glm-4",
    "messages": [
      {
        "role": "user",
        "content": "你好"
      }
    ],
    "stream": true,
    "temperature": 0.7,
    "max_tokens": 1000
  }'
```

流式响应格式（Server-Sent Events）：
```
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"你"},"index":0}],...}

data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"好"},"index":0}],...}

data: [DONE]
```

响应格式：直接透传目标 API 的响应

## 技术实现

### 1. 请求透传机制

本项目采用 **请求透传** 模式，使用 `map[string]any` 来解析和转发 JSON 数据，而不是使用固定的结构体。

**优点：**
- 🔄 **灵活性**：支持任意 JSON 字段，不受结构体限制
- 🚀 **扩展性**：目标 API 新增字段无需修改代码
- 🛠️ **可维护性**：后续可以在中间件层添加数据处理逻辑

**核心代码：**

```go
// handler/handler.go
var requestData map[string]any
c.ShouldBindJSON(&requestData)

// 获取 Authorization header
authHeader := c.GetHeader("Authorization")

// proxy/proxy.go
func (p *Proxy) ForwardRequest(requestData map[string]any, authHeader string) (map[string]any, error) {
    // 序列化请求
    requestBody, _ := json.Marshal(requestData)

    // 发送请求，透传 Authorization header
    targetReq.Header.Set("Authorization", authHeader)

    // 解析响应
    var responseData map[string]any
    json.Unmarshal(respBody, &responseData)
    return responseData, nil
}
```

### 2. API Key 透传机制

API Key 不再通过配置文件设置，而是通过请求的 `Authorization` header 直接透传给目标 API。

**优点：**
- 🔐 **安全性**：每个请求使用独立的 API Key，避免单点泄露风险
- 🎯 **灵活性**：支持多用户、多 API Key 场景
- 📊 **可追踪**：可以根据不同的 API Key 进行使用统计

## 配置说明

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| PORT | 服务监听端口 | 8080 |
| TARGET_API_URL | 目标 API 地址 | https://open.bigmodel.cn/api/coding/paas/v4 |
| LOG_LEVEL | 日志级别 | info |
| DEBUG | 是否启用 Debug 模式 | false |
| DEBUG_LOG_FILE | Debug 日志文件路径 | debug.json |

**注意**：不再需要配置 `TARGET_API_KEY`，API Key 通过请求 header 传递。

## 图片自动识别

**默认启用，无需配置**。当请求中包含图片时，自动识别并转换为文本描述。

### 主要特性

- **自动识别**：检测到 `image_url` 类型自动触发识别
- **智能缓存**：相同图片只识别一次，24小时有效期
- **文件持久化**：使用 BuntDB 缓存保存到 `image_cache.db`，重启后仍可用
- **性能优化**：缓存命中时响应速度提升 10 倍以上
- **成本优化**：避免重复调用 Vision API
- **依赖库**：使用 `github.com/gophertool/tool/db/cache` (BuntDB 实现)

### 使用示例

发送带图片的请求，无需任何特殊配置：

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "glm-4",
    "messages": [{
      "role": "user",
      "content": [
        {"type": "text", "text": "这张图片显示什么？"},
        {"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}}
      ]
    }]
  }'
```

**处理流程**：
1. 检测到图片，计算哈希
2. 查找缓存（如果有）
3. 识别图片（或使用缓存）
4. 替换为文本描述
5. 保存缓存

**日志输出**：
```
检测到图片，开始识别（哈希: a1b2c3d4e5f6...）...
图片识别成功，转换为文本
图片识别结果已缓存（哈希: a1b2c3d4e5f6...）
```

详见 [IMAGE_AUTO_RECOGNITION.md](IMAGE_AUTO_RECOGNITION.md)。

## Debug 模式

启用 Debug 模式后，所有的请求和响应数据会以 JSON 数组的形式保存到文件中，方便调试和问题排查。

**注意**：流式请求（`stream: true`）不会被记录到 Debug 日志中，因为内容量太大。

### 启用 Debug 模式

在 `.env` 文件中设置：

```env
DEBUG=true
DEBUG_LOG_FILE=debug.json
```

### Debug 日志格式

```json
[
  {
    "timestamp": "2024-01-01T12:00:00Z",
    "request": {
      "model": "glm-4",
      "messages": [
        {
          "role": "user",
          "content": "你好"
        }
      ],
      "temperature": 0.7
    },
    "response": {
      "id": "chatcmpl-xxx",
      "object": "chat.completion",
      "choices": [...]
    }
  },
  {
    "timestamp": "2024-01-01T12:01:00Z",
    "request": {...},
    "response": {...}
  }
]
```

### 注意事项

- Debug 日志会在每次请求后实时更新
- 日志文件可能包含敏感信息，请妥善保管
- 生产环境建议关闭 Debug 模式
- Debug 日志已添加到 `.gitignore`，不会被提交到版本控制

## 开发计划

- [x] 基础透传转发功能
- [x] Debug 日志记录
- [x] 流式响应支持（SSE）
- [x] 图片识别功能（Vision API）
- [x] 默认提示词支持（Vision API 可省略 prompt 参数）
- [x] 图片自动识别（默认启用，智能缓存，文件持久化）
- [ ] 请求/响应数据处理和转换
- [ ] 请求速率限制
- [ ] 错误重试机制
- [ ] 监控和指标

## 许可证

MIT
