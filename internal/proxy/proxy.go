package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"glm-tool/config"

	"github.com/gin-gonic/gin"
	"github.com/gophertool/tool/log"
)

type Proxy struct {
	targetURL string
	anthropicURL string
	client    *http.Client
}

func NewProxy() *Proxy {
	return &Proxy{
		targetURL: config.AppConfig.TargetAPIURL,
		anthropicURL: config.AppConfig.AnthropicAPIURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (p *Proxy) ForwardRequest(requestData map[string]any, authHeader string) (map[string]any, error) {
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	log.Infof("转发请求到: %s/chat/completions", p.targetURL)
	log.Infof("请求体: %s", string(requestBody))

	targetReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/chat/completions", p.targetURL),
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	targetReq.Header.Set("Content-Type", "application/json")
	// 直接透传客户端的 Authorization header
	targetReq.Header.Set("Authorization", authHeader)

	resp, err := p.client.Do(targetReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Infof("响应状态码: %d", resp.StatusCode)
	log.Infof("响应体: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("目标 API 返回错误 (状态码: %d): %s", resp.StatusCode, string(respBody))
	}

	var responseData map[string]any
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return responseData, nil
}

func (p *Proxy) ForwardGetRequest(endpoint string, authHeader string) (map[string]any, error) {
	targetURL := fmt.Sprintf("%s/%s", p.targetURL, endpoint)
	log.Infof("转发 GET 请求到: %s", targetURL)

	targetReq, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 直接透传客户端的 Authorization header
	targetReq.Header.Set("Authorization", authHeader)

	resp, err := p.client.Do(targetReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Infof("响应状态码: %d", resp.StatusCode)
	log.Infof("响应体: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("目标 API 返回错误 (状态码: %d): %s", resp.StatusCode, string(respBody))
	}

	var responseData map[string]any
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return responseData, nil
}

// ForwardStreamRequest 转发流式请求并流式返回响应
func (p *Proxy) ForwardStreamRequest(c *gin.Context, requestData map[string]any, authHeader string) error {
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	log.Infof("转发流式请求到: %s/chat/completions", p.targetURL)

	targetReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/chat/completions", p.targetURL),
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	targetReq.Header.Set("Content-Type", "application/json")
	targetReq.Header.Set("Authorization", authHeader)
	targetReq.Header.Set("Accept", "text/event-stream")

	// 发送请求
	resp, err := p.client.Do(targetReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("目标 API 返回错误 (状态码: %d): %s", resp.StatusCode, string(respBody))
	}

	// 设置响应头为 SSE 格式
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 创建一个 flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// 逐行读取并转发响应
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Infof("流式响应完成")
				break
			}
			return fmt.Errorf("读取流式响应失败: %w", err)
		}

		// 写入客户端
		if _, err := c.Writer.Write(line); err != nil {
			return fmt.Errorf("写入响应失败: %w", err)
		}

		// 立即刷新
		flusher.Flush()
	}

	return nil
}

// ForwardAnthropicRequest 转发 Anthropic 格式的请求
func (p *Proxy) ForwardAnthropicRequest(requestData map[string]any, authHeader string) (map[string]any, error) {
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	log.Infof("转发 Anthropic 请求到: %s/v1/messages", p.anthropicURL)
	log.Infof("请求体: %s", string(requestBody))

	targetReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/messages", p.anthropicURL),
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	targetReq.Header.Set("Content-Type", "application/json")
	targetReq.Header.Set("Authorization", authHeader)
	// Anthropic API 需要 x-api-key header 或者使用特定的 version
	targetReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(targetReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Infof("Anthropic 响应状态码: %d", resp.StatusCode)
	log.Infof("Anthropic 响应体: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("目标 API 返回错误 (状态码: %d): %s", resp.StatusCode, string(respBody))
	}

	var responseData map[string]any
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return responseData, nil
}

// ForwardAnthropicStreamRequest 转发 Anthropic 流式请求并流式返回响应
func (p *Proxy) ForwardAnthropicStreamRequest(c *gin.Context, requestData map[string]any, authHeader string) error {
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	log.Infof("转发 Anthropic 流式请求到: %s/v1/messages", p.anthropicURL)

	targetReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/messages", p.anthropicURL),
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	targetReq.Header.Set("Content-Type", "application/json")
	targetReq.Header.Set("Authorization", authHeader)
	targetReq.Header.Set("anthropic-version", "2023-06-01")
	targetReq.Header.Set("Accept", "text/event-stream")

	// 发送请求
	resp, err := p.client.Do(targetReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("目标 API 返回错误 (状态码: %d): %s", resp.StatusCode, string(respBody))
	}

	// 设置响应头为 SSE 格式
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 创建一个 flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// 逐行读取并转发响应
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Infof("Anthropic 流式响应完成")
				break
			}
			return fmt.Errorf("读取流式响应失败: %w", err)
		}

		// 写入客户端
		if _, err := c.Writer.Write(line); err != nil {
			return fmt.Errorf("写入响应失败: %w", err)
		}

		// 立即刷新
		flusher.Flush()
	}

	return nil
}

// ForwardAnthropicCountTokensRequest 转发 Anthropic Count Tokens 请求
func (p *Proxy) ForwardAnthropicCountTokensRequest(requestData map[string]any, authHeader string) (map[string]any, error) {
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	log.Infof("转发 Anthropic Count Tokens 请求到: %s/v1/messages/count_tokens", p.anthropicURL)
	log.Infof("请求体: %s", string(requestBody))

	targetReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/messages/count_tokens", p.anthropicURL),
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	targetReq.Header.Set("Content-Type", "application/json")
	targetReq.Header.Set("Authorization", authHeader)
	targetReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(targetReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Infof("Anthropic Count Tokens 响应状态码: %d", resp.StatusCode)
	log.Infof("Anthropic Count Tokens 响应体: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("目标 API 返回错误 (状态码: %d): %s", resp.StatusCode, string(respBody))
	}

	var responseData map[string]any
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return responseData, nil
}
