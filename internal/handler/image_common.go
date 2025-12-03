package handler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"glm-tool/internal/cache"
	"glm-tool/internal/vision"

	"github.com/gophertool/tool/log"
)

// ImageTask 通用图片识别任务
type ImageTask struct {
	ContentIndex int    // content 数组的索引
	Base64Data   string // 图片 base64 数据
	ImageHash    string // 图片哈希值
	ImageID      string // 图片 ID（自动生成）
}

// ImageResult 通用图片识别结果
type ImageResult struct {
	ContentIndex int    // content 数组的索引
	Text         string // 识别结果文本
	Success      bool   // 是否成功
}

// ImageReference 图片引用信息
type ImageReference struct {
	ImageID string // 图片 ID
	Number  int    // 编号（从文本中提取）
}

// extractImageReferences 从 content 中提取所有图片引用（按出现顺序）
// 返回：图片位置 -> 引用信息的映射
// msgIdx: 消息索引，用于生成 ID
// startCounter: 起始计数器值（当前消息内的图片序号）
func extractImageReferences(content []interface{}, msgIdx int, startCounter int) map[int]ImageReference {
	references := make(map[int]ImageReference)

	// 正则表达式：匹配 [Image #数字]（不带 ID 的）
	re := regexp.MustCompile(`\[Image\s*#(\d+)\]`)

	// 用于生成唯一的图片 ID
	imageIDCounter := startCounter

	// 遍历 content，按顺序处理
	for i, item := range content {
		contentItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		contentType, ok := contentItem["type"].(string)
		if !ok {
			continue
		}

		// 如果是图片类型（image 或 image_url）
		if contentType == "image" || contentType == "image_url" {
			// 为这张图片生成 ID，格式：#msgIdx_imageIdx
			imageID := fmt.Sprintf("#%d_%d", msgIdx, imageIDCounter)
			imageIDCounter++

			// 在后续的**第一个**文本中查找对应的引用编号
			foundNumber := 0
			for j := i + 1; j < len(content); j++ {
				nextItem, ok := content[j].(map[string]interface{})
				if !ok {
					continue
				}

				nextType, ok := nextItem["type"].(string)
				if !ok {
					continue
				}

				// 只在紧跟的第一个文本消息中查找编号
				if nextType == "text" {
					if text, ok := nextItem["text"].(string); ok {
						// 查找第一个匹配的编号
						matches := re.FindStringSubmatch(text)
						if len(matches) > 1 {
							if num, err := strconv.Atoi(matches[1]); err == nil {
								foundNumber = num
								break
							}
						}
					}
					break // 只检查紧跟的第一个文本消息
				}
			}

			references[i] = ImageReference{
				ImageID: imageID,
				Number:  foundNumber,
			}
		}
	}

	return references
}

// fillImageIDsInText 将文本中的 [Image #数字] 替换为带 ID 的格式 [Image #ID]
// 只替换当前图片到下一个图片之间的文本
func fillImageIDsInText(content []interface{}, references map[int]ImageReference) {
	// 正则表达式：匹配 [Image #数字]（不带 ID 的）
	re := regexp.MustCompile(`\[Image\s*#\d+\]`)

	for imageIndex, ref := range references {
		// 找到下一个图片的位置
		nextImageIndex := len(content)
		for i := imageIndex + 1; i < len(content); i++ {
			if item, ok := content[i].(map[string]interface{}); ok {
				if contentType, ok := item["type"].(string); ok {
					if contentType == "image" || contentType == "image_url" {
						nextImageIndex = i
						break
					}
				}
			}
		}

		// 新的标识符格式：[Image #ID]
		newIdentifier := fmt.Sprintf("[Image %s]", ref.ImageID)

		// 在 imageIndex 到 nextImageIndex 之间的文本中替换
		for i := imageIndex + 1; i < nextImageIndex; i++ {
			contentItem, ok := content[i].(map[string]interface{})
			if !ok {
				continue
			}

			if contentType, ok := contentItem["type"].(string); ok && contentType == "text" {
				if text, ok := contentItem["text"].(string); ok {
					// 替换所有 [Image #数字] 为 [Image #ID]
					newText := re.ReplaceAllString(text, newIdentifier)
					if newText != text {
						content[i] = map[string]interface{}{
							"type": "text",
							"text": newText,
						}
						log.Debugf("填充 ID: 将 Content[%d] 中的图片引用替换为 %s", i, newIdentifier)
					}
				}
			}
		}
	}
}

// fillImageIDsInTextWithIDs 在文本中填充图片 ID
// 按照 [Image #数字] 中的数字作为索引，从 imageIDs 数组中获取对应的 ID
// 例如：[Image #1] -> imageIDs[0], [Image #2] -> imageIDs[1]
func fillImageIDsInTextWithIDs(content []interface{}, imageIDs []string) {
	// 正则表达式：匹配 [Image #数字]（捕获数字）
	re := regexp.MustCompile(`\[Image\s*#(\d+)\]`)

	for i, item := range content {
		contentItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if contentType, ok := contentItem["type"].(string); ok && contentType == "text" {
			if text, ok := contentItem["text"].(string); ok {
				// 替换所有 [Image #数字] 为 [Image #ID]
				newText := re.ReplaceAllStringFunc(text, func(match string) string {
					// 提取数字
					matches := re.FindStringSubmatch(match)
					if len(matches) > 1 {
						if num, err := strconv.Atoi(matches[1]); err == nil {
							// 数字从 1 开始，数组从 0 开始
							idx := num - 1
							if idx >= 0 && idx < len(imageIDs) {
								newID := fmt.Sprintf("[Image %s]", imageIDs[idx])
								log.Debugf("填充 ID: [Image #%d] -> %s", num, newID)
								return newID
							}
						}
					}
					return match // 保持原样
				})

				if newText != text {
					content[i] = map[string]interface{}{
						"type": "text",
						"text": newText,
					}
				}
			}
		}
	}
}

// fillImageIDsInTextWithCurrentID 在文本中填充图片 ID
// 将所有 [Image #任意数字] 替换为当前消息对应的图片 ID
// 这样无论用户写 [Image #1] 还是 [Image #2]，都会替换为当前活跃的图片 ID
func fillImageIDsInTextWithCurrentID(content []interface{}, currentImageID string) {
	// 正则表达式：匹配 [Image #数字]
	re := regexp.MustCompile(`\[Image\s*#\d+\]`)

	newIdentifier := fmt.Sprintf("[Image %s]", currentImageID)

	for i, item := range content {
		contentItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if contentType, ok := contentItem["type"].(string); ok && contentType == "text" {
			if text, ok := contentItem["text"].(string); ok {
				// 替换所有 [Image #数字] 为 [Image #currentImageID]
				newText := re.ReplaceAllString(text, newIdentifier)
				if newText != text {
					log.Debugf("填充 ID: 替换为 %s", newIdentifier)
					content[i] = map[string]interface{}{
						"type": "text",
						"text": newText,
					}
				}
			}
		}
	}
}

// fillImageIDsInTextByNumber 在文本中按编号填充图片 ID
// [Image #1] -> activeImageIDs[0], [Image #2] -> activeImageIDs[1], ...
// 如果编号超出范围，保持原样不替换
func fillImageIDsInTextByNumber(content []interface{}, activeImageIDs []string) {
	// 正则表达式：匹配 [Image #数字]（捕获数字）
	re := regexp.MustCompile(`\[Image\s*#(\d+)\]`)

	for i, item := range content {
		contentItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if contentType, ok := contentItem["type"].(string); ok && contentType == "text" {
			if text, ok := contentItem["text"].(string); ok {
				// 替换所有 [Image #数字] 为对应的 [Image #ID]
				newText := re.ReplaceAllStringFunc(text, func(match string) string {
					// 提取数字
					matches := re.FindStringSubmatch(match)
					if len(matches) > 1 {
						if num, err := strconv.Atoi(matches[1]); err == nil {
							// 数字从 1 开始，数组从 0 开始
							idx := num - 1
							if idx >= 0 && idx < len(activeImageIDs) {
								newID := fmt.Sprintf("[Image %s]", activeImageIDs[idx])
								log.Debugf("填充 ID: [Image #%d] -> %s", num, newID)
								return newID
							}
							// 超出范围，跳过不替换
							log.Debugf("填充 ID: [Image #%d] 超出范围（只有 %d 张图片），跳过", num, len(activeImageIDs))
						}
					}
					return match // 保持原样
				})

				if newText != text {
					content[i] = map[string]interface{}{
						"type": "text",
						"text": newText,
					}
				}
			}
		}
	}
}

// buildImagePrefix 根据图片引用构建前缀
func buildImagePrefix(ref ImageReference) string {
	// ID 格式：#msgIdx_imageIdx，如 #0_1
	return fmt.Sprintf("[Image %s] 以下是系统自动识别的图片内容描述：\n\n", ref.ImageID)
}

// processImageWithCache 检查缓存并处理图片（如果命中缓存）
// 返回：是否命中缓存、识别结果文本
func processImageWithCache(imageHash string, ref ImageReference, prefix string) (bool, string) {
	if cachedResult, found := cache.GetImageResult(imageHash); found {
		if ref.Number > 0 {
			log.Infof("使用缓存的图片识别结果（哈希: %s, ID: %s, 编号: #%d）", imageHash[:16], ref.ImageID, ref.Number)
		} else {
			log.Infof("使用缓存的图片识别结果（哈希: %s, ID: %s）", imageHash[:16], ref.ImageID)
		}
		return true, prefix + cachedResult
	}
	return false, ""
}

// recognizeImagesConcurrently 并发识别多张图片
func recognizeImagesConcurrently(tasks []ImageTask, apiKey string, apiType string) []ImageResult {
	if len(tasks) == 0 {
		return nil
	}

	log.Infof("%s检测到 %d 张图片需要识别，开始并发处理...", apiType, len(tasks))

	var wg sync.WaitGroup
	resultChan := make(chan ImageResult, len(tasks))

	for _, task := range tasks {
		wg.Add(1)
		go func(t ImageTask) {
			defer wg.Done()

			log.Infof("开始识别图片（哈希: %s, ID: %s）...", t.ImageHash[:16], t.ImageID)

			// 构建识别请求
			visionReq := vision.ImageAnalysisRequest{
				ImageBase64: t.Base64Data,
				Prompt:      "", // 使用默认 prompt
				APIKey:      apiKey,
			}

			// 调用识别
			result, err := vision.AnalyzeImage(visionReq)
			if err != nil {
				log.Warnf("图片识别失败（哈希: %s, ID: %s）: %v", t.ImageHash[:16], t.ImageID, err)
				resultChan <- ImageResult{
					ContentIndex: t.ContentIndex,
					Success:      false,
				}
				return
			}

			if !result.Success {
				log.Warnf("图片识别失败（哈希: %s, ID: %s）: %s", t.ImageHash[:16], t.ImageID, result.Error)
				resultChan <- ImageResult{
					ContentIndex: t.ContentIndex,
					Success:      false,
				}
				return
			}

			// 识别成功
			log.Infof("图片识别成功（哈希: %s, ID: %s），转换为文本", t.ImageHash[:16], t.ImageID)

			// 保存到缓存
			cache.SetImageResult(t.ImageHash, result.Data)
			log.Infof("图片识别结果已缓存（哈希: %s, ID: %s）", t.ImageHash[:16], t.ImageID)

			// 构建带 ID 的识别结果（前缀在外部构建）
			resultChan <- ImageResult{
				ContentIndex: t.ContentIndex,
				Text:         result.Data,
				Success:      true,
			}
		}(task)
	}

	// 等待所有识别完成
	wg.Wait()
	close(resultChan)

	log.Infof("%s所有图片识别完成", apiType)

	// 收集结果
	var results []ImageResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// applyRecognitionResults 将识别结果应用到 content
// 1. 将图片本身替换为识别结果文本
// 2. 在图片后面、下一个图片之前的所有文本消息中，将所有 [Image #ID] 引用替换为该图片的识别结果
func applyRecognitionResults(content []interface{}, results []ImageResult, references map[int]ImageReference) {
	// 第一步：替换图片本身为识别结果
	imageResults := make(map[int]string) // key: 图片索引, value: 完整识别结果
	imageIDs := make(map[int]string)      // key: 图片索引, value: 图片ID
	for _, result := range results {
		if result.Success {
			// 获取对应的引用信息
			ref := references[result.ContentIndex]
			prefix := buildImagePrefix(ref)
			fullText := prefix + result.Text

			// 替换图片为文本
			content[result.ContentIndex] = map[string]interface{}{
				"type": "text",
				"text": fullText,
			}

			log.Infof("将 Content[%d] 从图片替换为识别结果文本 (ID: %s)", result.ContentIndex, ref.ImageID)

			// 记录这个图片的识别结果和ID
			imageResults[result.ContentIndex] = fullText
			imageIDs[result.ContentIndex] = ref.ImageID
		}
		// 识别失败的保持原样，不做修改
	}

	// 第二步：在每个图片后面、下一个图片之前的文本中，替换所有包含该图片ID的引用
	for imageIndex, recognitionResult := range imageResults {
		imageID := imageIDs[imageIndex]

		// 构建要替换的标识符：[Image #ID]，如 [Image #0_1]
		identifier := fmt.Sprintf("[Image %s]", imageID)

		// 找到下一个图片的位置
		nextImageIndex := len(content) // 默认到末尾
		for i := imageIndex + 1; i < len(content); i++ {
			if item, ok := content[i].(map[string]interface{}); ok {
				if contentType, ok := item["type"].(string); ok {
					if contentType == "image" || contentType == "image_url" {
						nextImageIndex = i
						break
					}
				}
			}
		}

		log.Infof("图片 ID: %s 的替换范围: Content[%d] 到 Content[%d]", imageID, imageIndex+1, nextImageIndex-1)

		// 在 imageIndex 到 nextImageIndex 之间的文本消息中，替换所有包含该图片ID的引用
		replacedCount := 0
		for i := imageIndex + 1; i < nextImageIndex; i++ {
			contentItem, ok := content[i].(map[string]interface{})
			if !ok {
				continue
			}

			if contentType, ok := contentItem["type"].(string); ok && contentType == "text" {
				if text, ok := contentItem["text"].(string); ok {
					// 替换所有 [Image #ID] 为识别结果
					if strings.Contains(text, identifier) {
						newText := strings.ReplaceAll(text, identifier, recognitionResult)
						replacedCount += strings.Count(text, identifier)
						log.Infof("  在 Content[%d] 中找到匹配的引用 '%s'，替换为识别结果", i, identifier)
						content[i] = map[string]interface{}{
							"type": "text",
							"text": newText,
						}
					}
				}
			}
		}

		if replacedCount > 0 {
			log.Infof("图片 ID: %s 在后续文本中共替换了 %d 处引用", imageID, replacedCount)
		}
	}
}

// extractBase64FromURL 从 data URI 或直接的 base64 字符串中提取 base64 数据
func extractBase64FromURL(url string) string {
	// 如果是 data URI 格式（data:image/xxx;base64,xxxxx）
	if strings.HasPrefix(url, "data:") {
		parts := strings.SplitN(url, ",", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// 否则假定就是 base64 数据
	return url
}

// extractBase64FromData 从 data URI 或直接的 base64 字符串中提取 base64 数据（别名）
func extractBase64FromData(data string) string {
	return extractBase64FromURL(data)
}
