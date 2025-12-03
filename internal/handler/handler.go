package handler

import (
	"net/http"

	"glm-tool/internal/debuglog"
	"glm-tool/internal/proxy"

	"github.com/gin-gonic/gin"
	"github.com/gophertool/tool/log"
)

type Handler struct {
	proxy *proxy.Proxy
}

func NewHandler() *Handler {
	return &Handler{
		proxy: proxy.NewProxy(),
	}
}

func (h *Handler) ChatCompletions(c *gin.Context) {
	var requestData map[string]any

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Warnf("解析请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "无效的请求格式",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// 获取请求中的 Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		log.Warnf("缺少 Authorization header")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "缺少 API Key",
				"type":    "authentication_error",
			},
		})
		return
	}

	// log.Printf("收到请求: %v", requestData)

	// [调试中间件] 识别图片并转换为文本（仅在 DEBUG 模式下生效）
	if err := ProcessImageToText(requestData, authHeader); err != nil {
		log.Warnf("图片处理失败: %v", err)
	}

	// 检查是否为流式请求
	isStream := false
	if stream, ok := requestData["stream"].(bool); ok && stream {
		isStream = true
	}

	if isStream {
		// 流式响应：直接透传
		log.Infof("处理流式请求")
		err := h.proxy.ForwardStreamRequest(c, requestData, authHeader)
		if err != nil {
			log.Warnf("转发流式请求失败: %v", err)
			// 流式响应错误时，尝试发送错误消息
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": err.Error(),
					"type":    "proxy_error",
				},
			})
		}
		// 流式请求不记录 debug 日志（内容太大）
	} else {
		// 非流式响应：正常处理
		respData, err := h.proxy.ForwardRequest(requestData, authHeader)

		// 记录 debug 日志
		debuglog.LogRequest(requestData, respData, err)

		if err != nil {
			log.Warnf("转发请求失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": err.Error(),
					"type":    "proxy_error",
				},
			})
			return
		}

		c.JSON(http.StatusOK, respData)
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "glm-tool",
	})
}

func (h *Handler) ListModels(c *gin.Context) {
	// 获取请求中的 Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		log.Warnf("缺少 Authorization header")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "缺少 API Key",
				"type":    "authentication_error",
			},
		})
		return
	}

	log.Infof("收到 models 列表请求")

	respData, err := h.proxy.ForwardGetRequest("models", authHeader)

	// 记录 debug 日志
	debuglog.LogRequest(nil, respData, err)

	if err != nil {
		log.Warnf("转发请求失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "proxy_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, respData)
}

func (h *Handler) AnthropicMessages(c *gin.Context) {
	var requestData map[string]any

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Warnf("解析 Anthropic 请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "无效的请求格式",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// 获取请求中的 Authorization header 或 x-api-key header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		// Anthropic API 也支持 x-api-key header
		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" {
			authHeader = apiKey
		}
	}

	if authHeader == "" {
		log.Warnf("缺少 Authorization 或 x-api-key header")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "缺少 API Key",
				"type":    "authentication_error",
			},
		})
		return
	}

	// [调试中间件] 识别图片并转换为文本（仅在 DEBUG 模式下生效）
	if err := ProcessImageToTextForAnthropic(requestData, authHeader); err != nil {
		log.Warnf("Anthropic 图片处理失败: %v", err)
	}

	// 检查是否为流式请求
	isStream := false
	if stream, ok := requestData["stream"].(bool); ok && stream {
		isStream = true
	}

	if isStream {
		// 流式响应：直接透传
		log.Infof("处理 Anthropic 流式请求")
		err := h.proxy.ForwardAnthropicStreamRequest(c, requestData, authHeader)
		if err != nil {
			log.Warnf("转发 Anthropic 流式请求失败: %v", err)
			// 流式响应错误时，尝试发送错误消息
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": err.Error(),
					"type":    "proxy_error",
				},
			})
		}
	} else {
		// 非流式响应：正常处理
		respData, err := h.proxy.ForwardAnthropicRequest(requestData, authHeader)

		// 记录 debug 日志
		debuglog.LogRequest(requestData, respData, err)

		if err != nil {
			log.Warnf("转发 Anthropic 请求失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": err.Error(),
					"type":    "proxy_error",
				},
			})
			return
		}

		c.JSON(http.StatusOK, respData)
	}
}


func (h *Handler) AnthropicCountTokens(c *gin.Context) {
	var requestData map[string]any

	if err := c.ShouldBindJSON(&requestData); err != nil {
		log.Warnf("解析 Anthropic Count Tokens 请求失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "无效的请求格式",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// 获取请求中的 Authorization header 或 x-api-key header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		// Anthropic API 也支持 x-api-key header
		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" {
			authHeader = apiKey
		}
	}

	if authHeader == "" {
		log.Warnf("缺少 Authorization 或 x-api-key header")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "缺少 API Key",
				"type":    "authentication_error",
			},
		})
		return
	}

	// 转发请求
	respData, err := h.proxy.ForwardAnthropicCountTokensRequest(requestData, authHeader)

	// 记录 debug 日志
	debuglog.LogRequest(requestData, respData, err)

	if err != nil {
		log.Warnf("转发 Anthropic Count Tokens 请求失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "proxy_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, respData)
}
