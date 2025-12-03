package handler

import (
	"strings"

	"glm-tool/internal/cache"

	"github.com/gophertool/tool/log"
)

// collectAnthropicImageTasks 从 Anthropic 格式的 content 中收集图片任务
func collectAnthropicImageTasks(content []interface{}, references map[int]ImageReference) []ImageTask {
	var tasks []ImageTask

	for i, item := range content {
		contentItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// 检查 type 是否为 image
		if contentType, ok := contentItem["type"].(string); ok && contentType == "image" {
			// 获取 source 对象
			if source, ok := contentItem["source"].(map[string]interface{}); ok {
				// 提取图片数据
				if mediaType, ok := source["media_type"].(string); ok && strings.HasPrefix(mediaType, "image/") {
					if data, ok := source["data"].(string); ok {
						// 提取 base64 数据
						base64Data := extractBase64FromData(data)
						imageHash := cache.ComputeHash(base64Data)

						// 获取这张图片的引用信息
						ref, hasRef := references[i]
						if !hasRef {
							continue // 没有引用信息，跳过
						}

						// 构建前缀
						prefix := buildImagePrefix(ref)

						// 检查缓存
						if hit, text := processImageWithCache(imageHash, ref, prefix); hit {
							content[i] = map[string]interface{}{
								"type": "text",
								"text": text,
							}
							continue
						}

						// 添加到待处理任务列表
						tasks = append(tasks, ImageTask{
							ContentIndex: i,
							Base64Data:   base64Data,
							ImageHash:    imageHash,
							ImageID:      ref.ImageID,
						})
					}
				}
			}
		}
	}

	return tasks
}

// ProcessImageToTextForAnthropic 专为 Anthropic API 处理图片：并发识别图片并转换为文本
func ProcessImageToTextForAnthropic(requestData map[string]any, authHeader string) error {
	// 提取 API Key
	apiKey := strings.TrimPrefix(authHeader, "Bearer ")

	// 获取 messages 字段
	messages, ok := requestData["messages"].([]interface{})
	if !ok {
		return nil
	}

	log.Infof("Anthropic API 开始处理请求，共 %d 条消息", len(messages))

	// 全局图片计数器，确保整个请求中的图片 ID 唯一
	globalImageCounter := 1

	// 存储所有消息的 content 和 references
	type messageData struct {
		content         []interface{}
		references      map[int]ImageReference
		activeImageIDs  []string // 当前消息对应的活跃图片 ID 列表（按顺序）
	}
	allMessages := make([]messageData, 0)

	// 当前活跃的图片 ID 列表（遇到新图片时更新）
	currentActiveImageIDs := make([]string, 0)

	// 第一遍：提取所有图片引用，确定每个消息对应的图片 ID 列表
	for msgIdx, msg := range messages {
		message, ok := msg.(map[string]interface{})
		if !ok {
			// 复制当前活跃列表
			activeIDs := make([]string, len(currentActiveImageIDs))
			copy(activeIDs, currentActiveImageIDs)
			allMessages = append(allMessages, messageData{activeImageIDs: activeIDs})
			continue
		}

		// 打印消息角色
		if role, ok := message["role"].(string); ok {
			log.Infof("处理消息 #%d (role: %s)", msgIdx, role)
		}

		// 获取 content 字段
		content, ok := message["content"].([]interface{})
		if !ok {
			activeIDs := make([]string, len(currentActiveImageIDs))
			copy(activeIDs, currentActiveImageIDs)
			allMessages = append(allMessages, messageData{activeImageIDs: activeIDs})
			continue
		}

		log.Infof("消息 #%d 包含 %d 个 content 项", msgIdx, len(content))

		// 打印 content 结构（用于调试）
		for i, item := range content {
			if contentItem, ok := item.(map[string]interface{}); ok {
				if contentType, ok := contentItem["type"].(string); ok {
					if contentType == "text" {
						if text, ok := contentItem["text"].(string); ok {
							log.Infof("  Content[%d]: type=text, text=%s", i, truncateString(text, 100))
						}
					} else if contentType == "image" {
						// 提取并打印图片哈希值
						if source, ok := contentItem["source"].(map[string]interface{}); ok {
							if data, ok := source["data"].(string); ok {
								base64Data := extractBase64FromData(data)
								imageHash := cache.ComputeHash(base64Data)
								log.Infof("  Content[%d]: type=image, hash=%s", i, imageHash[:16])
							}
						}
					}
				}
			}
		}

		// 提取图片引用
		references := extractImageReferences(content, msgIdx, globalImageCounter)
		if len(references) > 0 {
			log.Infof("Anthropic API 提取到 %d 个图片引用", len(references))
			// 清空当前活跃列表，重新填充（新的图片组）
			currentActiveImageIDs = make([]string, 0)
			for idx, ref := range references {
				log.Infof("  图片[%d] -> ID: %s, 编号: #%d", idx, ref.ImageID, ref.Number)
				currentActiveImageIDs = append(currentActiveImageIDs, ref.ImageID)
			}
			// 更新全局计数器
			globalImageCounter += len(references)
		}

		// 复制当前活跃列表
		activeIDs := make([]string, len(currentActiveImageIDs))
		copy(activeIDs, currentActiveImageIDs)
		allMessages = append(allMessages, messageData{
			content:        content,
			references:     references,
			activeImageIDs: activeIDs,
		})
	}

	// 第二遍：在所有消息的文本中填充 ID（使用每个消息对应的活跃图片列表）
	for msgIdx, msgData := range allMessages {
		if msgData.content == nil || len(msgData.activeImageIDs) == 0 {
			continue
		}
		// 填充文本中的图片 ID（按编号匹配）
		fillImageIDsInTextByNumber(msgData.content, msgData.activeImageIDs)
		log.Debugf("消息 #%d 填充 ID 完成 (活跃图片: %v)", msgIdx, msgData.activeImageIDs)
	}

	// 第三遍：处理图片识别
	for msgIdx, msgData := range allMessages {
		if msgData.content == nil || len(msgData.references) == 0 {
			continue
		}

		// 收集图片任务
		tasks := collectAnthropicImageTasks(msgData.content, msgData.references)

		// 并发识别图片
		results := recognizeImagesConcurrently(tasks, apiKey, "Anthropic API ")

		// 应用识别结果
		applyRecognitionResults(msgData.content, results, msgData.references)

		log.Infof("消息 #%d 图片识别完成", msgIdx)
	}

	log.Infof("Anthropic API 请求处理完成")
	return nil
}
