package main

import (
	"glm-tool/config"
	"glm-tool/internal/handler"

	"github.com/gin-gonic/gin"
	"github.com/gophertool/tool/log"
)

func main() {
	config.LoadConfig()

	r := gin.Default()

	h := handler.NewHandler()

	r.GET("/health", h.HealthCheck)

	v1 := r.Group("/v1")
	{
		v1.POST("/chat/completions", h.ChatCompletions)
		v1.GET("/models", h.ListModels)
		v1.POST("/messages", h.AnthropicMessages)
		v1.POST("/messages/count_tokens", h.AnthropicCountTokens)
	}

	log.Infof("服务启动在端口: %s", config.AppConfig.Port)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Errorf("服务启动失败: %v", err)
	}
}
