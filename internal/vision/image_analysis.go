package vision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ImageAnalysisRequest 图片分析请求
type ImageAnalysisRequest struct {
	ImageBase64 string `json:"image_base64"` // 图片的 base64 编码
	Prompt      string `json:"prompt"`       // 分析提示词
	APIKey      string `json:"-"`            // API Key（不序列化）
}

// ImageAnalysisResponse 图片分析响应
type ImageAnalysisResponse struct {
	Success bool   `json:"success"`
	Data    string `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// VisionConfig 视觉模型配置
type VisionConfig struct {
	BaseURL     string
	Model       string
	Temperature float64
	TopP        float64
	MaxTokens   int
	Timeout     time.Duration
}

// DefaultVisionConfig 默认配置（完全按照 MCP 配置）
var DefaultVisionConfig = VisionConfig{
	BaseURL:     "https://open.bigmodel.cn/api/paas/v4",
	Model:       "glm-4.6v", // MCP 默认使用 glm-4.6v
	Temperature: 0.8,
	TopP:        0.6,
	MaxTokens:   16384,
	Timeout:     300 * time.Second,
}

// Thinking 思考配置（MCP 中使用）
type Thinking struct {
	Type string `json:"type"`
}

// ChatMessage OpenAI 格式的消息
type ChatMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

// ChatCompletionRequest OpenAI 格式的请求（完全按照 MCP）
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Thinking    *Thinking     `json:"thinking"` // MCP 使用这个参数
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
	TopP        float64       `json:"top_p"`
	MaxTokens   int           `json:"max_tokens"`
}

// ChatCompletionResponse OpenAI 格式的响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择项
type Choice struct {
	Index        int            `json:"index"`
	Message      MessageContent `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

// MessageContent 消息内容
type MessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage token 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// AnalyzeImage 分析图片
func AnalyzeImage(request ImageAnalysisRequest) (*ImageAnalysisResponse, error) {
	return AnalyzeImageWithConfig(request, DefaultVisionConfig)
}

// AnalyzeImageWithConfig 使用自定义配置分析图片（完全按照 MCP 实现）
func AnalyzeImageWithConfig(request ImageAnalysisRequest, config VisionConfig) (*ImageAnalysisResponse, error) {
	// 验证输入
	if request.ImageBase64 == "" {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   "image_base64 is required",
		}, fmt.Errorf("image_base64 is required")
	}

	// 如果没有提供 Prompt，使用默认的全面描述提示词
	if request.Prompt == "" {
		request.Prompt = "请详细全面地描述这张图片的内容，包括但不限于：\n" +
			"1. 整体场景和环境（室内/室外、时间、天气等）\n" +
			"2. 主要物体和人物（位置、大小、特征、动作、表情等）\n" +
			"3. 颜色搭配和光影效果\n" +
			"4. 构图和布局（前景、中景、背景）\n" +
			"5. 文字内容（如果图片中包含文字，请完整识别并提取）\n" +
			"6. 整体氛围、情绪和风格\n" +
			"7. 其他值得注意的细节\n\n" +
			"请用清晰、结构化的方式组织描述，确保信息准确完整。"
	}

	if request.APIKey == "" {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   "API key is required",
		}, fmt.Errorf("API key is required")
	}

	// 构建图片内容（支持 base64）
	imageURL := request.ImageBase64
	if len(imageURL) > 0 && imageURL[:5] != "data:" {
		// 如果不是 data: 开头，添加前缀
		imageURL = "data:image/jpeg;base64," + imageURL
	}

	// 构建消息（完全按照 MCP 的 createMultiModalMessage）
	messages := []ChatMessage{
		{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]string{
						"url": imageURL,
					},
				},
				map[string]interface{}{
					"type": "text",
					"text": request.Prompt,
				},
			},
		},
	}

	// 构建请求体（完全按照 MCP 的 visionCompletions）
	reqBody := ChatCompletionRequest{
		Model:    config.Model,
		Messages: messages,
		Thinking: &Thinking{
			Type: "enabled",
		},
		Stream:      false,
		Temperature: config.Temperature,
		TopP:        config.TopP,
		MaxTokens:   config.MaxTokens,
	}

	// 序列化请求
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to marshal request: %v", err),
		}, err
	}

	// 创建 HTTP 请求
	url := fmt.Sprintf("%s/chat/completions", config.BaseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create request: %v", err),
		}, err
	}

	// 设置请求头（完全按照 MCP 的 chatCompletions）
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", request.APIKey))
	httpReq.Header.Set("X-Title", "4.6V MCP Local")   // 与 MCP 一致
	httpReq.Header.Set("Accept-Language", "en-US,en") // 与 MCP 一致

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to send request: %v", err),
		}, err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to read response: %v", err),
		}, err
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(respBody)),
		}, fmt.Errorf("API error: %s", string(respBody))
	}

	// 解析响应
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to parse response: %v", err),
		}, err
	}

	// 提取结果（按照 MCP 的逻辑）
	if len(chatResp.Choices) == 0 {
		return &ImageAnalysisResponse{
			Success: false,
			Error:   "no choices in response",
		}, fmt.Errorf("no choices in response")
	}

	result := chatResp.Choices[0].Message.Content

	return &ImageAnalysisResponse{
		Success: true,
		Data:    result,
	}, nil
}
