package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"Zai/internal/config"
	"Zai/internal/model"
	"Zai/internal/upstream"
	"Zai/internal/util"
)

func HandleOptions(w http.ResponseWriter, r *http.Request) {
	util.SetCORSHeaders(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func HandleModels(w http.ResponseWriter, r *http.Request) {
	util.SetCORSHeaders(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	response := model.ModelsResponse{
		Object: "list",
		Data: []model.Model{
			{
				ID:      config.MODEL_NAME,
				Object:  "model",
				Created: time.Now().Unix(),
				OwnedBy: "z.ai",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	util.SetCORSHeaders(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	util.DebugLog("收到chat completions请求")

	// 验证API Key
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		util.DebugLog("缺少或无效的Authorization头")
		http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
		return
	}

	apiKey := strings.TrimPrefix(authHeader, "Bearer ")
	if apiKey != config.DEFAULT_KEY {
		util.DebugLog("无效的API key: %s", apiKey)
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	util.DebugLog("API key验证通过")

	// 解析请求
	var req model.OpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.DebugLog("JSON解析失败: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	util.DebugLog("请求解析成功 - 模型: %s, 流式: %v, 消息数: %d", req.Model, req.Stream, len(req.Messages))

	// 生成会话相关ID
	chatID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
	msgID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 构造上游请求
	upstreamReq := model.UpstreamRequest{
		Stream:   true, // 总是使用流式从上游获取
		ChatID:   chatID,
		ID:       msgID,
		Model:    "0727-360B-API", // 上游实际模型ID
		Messages: req.Messages,
		Params:   map[string]interface{}{},
		Features: map[string]interface{}{
			"enable_thinking": true,
		},
		BackgroundTasks: map[string]bool{
			"title_generation": false,
			"tags_generation":  false,
		},
		MCPServers: []string{},
		ModelItem: struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			OwnedBy string `json:"owned_by"`
		}{ID: "0727-360B-API", Name: "GLM-4.5", OwnedBy: "openai"},
		ToolServers: []string{},
		Variables: map[string]string{
			"{{USER_NAME}}":        "User",
			"{{USER_LOCATION}}":    "Unknown",
			"{{CURRENT_DATETIME}}": time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	// 选择本次对话使用的token
	authToken := config.UPSTREAM_TOKEN
	if config.ANON_TOKEN_ENABLED {
		if t, err := upstream.GetAnonymousToken(); err == nil {
			authToken = t
			util.DebugLog("匿名token获取成功: %s...", func() string {
				if len(t) > 10 {
					return t[:10]
				}
				return t
			}())
		} else {
			util.DebugLog("匿名token获取失败，回退固定token: %v", err)
		}
	}

	// 调用上游API
	if req.Stream {
		handleStreamResponseWithIDs(w, upstreamReq, chatID, authToken)
	} else {
		handleNonStreamResponseWithIDs(w, upstreamReq, chatID, authToken)
	}
}

func handleStreamResponseWithIDs(w http.ResponseWriter, upstreamReq model.UpstreamRequest, chatID string, authToken string) {
	util.DebugLog("开始处理流式响应 (chat_id=%s)", chatID)

	resp, err := upstream.CallUpstreamWithHeaders(upstreamReq, chatID, authToken)
	if err != nil {
		util.DebugLog("调用上游失败: %v", err)
		http.Error(w, "Failed to call upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		util.DebugLog("上游返回错误状态: %d", resp.StatusCode)
		// 读取错误响应体
		if config.DEBUG_MODE {
			body, _ := io.ReadAll(resp.Body)
			util.DebugLog("上游错误响应: %s", string(body))
		}
		http.Error(w, "Upstream error", http.StatusBadGateway)
		return
	}

	// 用于策略2：总是展示thinking（配合标签处理）
	transformThinking := func(s string) string {
		// 去 <summary>…</summary>
		s = regexp.MustCompile(`(?s)<summary>.*?</summary>`).ReplaceAllString(s, "")
		// 清理残留自定义标签，如 </thinking>、<Full> 等
		s = strings.ReplaceAll(s, "</thinking>", "")
		s = strings.ReplaceAll(s, "<Full>", "")
		s = strings.ReplaceAll(s, "</Full>", "")
		s = strings.TrimSpace(s)
		switch config.THINK_TAGS_MODE {
		case "think":
			s = regexp.MustCompile(`<details[^>]*>`).ReplaceAllString(s, "<think>")
			s = strings.ReplaceAll(s, "</details>", "</think>")
		case "strip":
			s = regexp.MustCompile(`<details[^>]*>`).ReplaceAllString(s, "")
			s = strings.ReplaceAll(s, "</details>", "")
		}
		// 处理每行前缀 "> "（包括起始位置）
		s = strings.TrimPrefix(s, "> ")
		s = strings.ReplaceAll(s, "\n> ", "\n")
		return strings.TrimSpace(s)
	}

	// 设置SSE头部
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 发送第一个chunk（role）
	firstChunk := model.OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   config.MODEL_NAME,
		Choices: []model.Choice{
			{
				Index: 0,
				Delta: model.Delta{Role: "assistant"},
			},
		},
	}
	writeSSEChunk(w, firstChunk)
	flusher.Flush()

	// 读取上游SSE流
	util.DebugLog("开始读取上游SSE流")
	scanner := bufio.NewScanner(resp.Body)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "" {
			continue
		}

		util.DebugLog("收到SSE数据 (第%d行): %s", lineCount, dataStr)

		var upstreamData model.UpstreamData
		if err := json.Unmarshal([]byte(dataStr), &upstreamData); err != nil {
			util.DebugLog("SSE数据解析失败: %v", err)
			continue
		}

		// 错误检测（data.error 或 data.data.error 或 顶层error）
		if (upstreamData.Error != nil) || (upstreamData.Data.Error != nil) || (upstreamData.Data.Inner != nil && upstreamData.Data.Inner.Error != nil) {
			errObj := upstreamData.Error
			if errObj == nil {
				errObj = upstreamData.Data.Error
			}
			if errObj == nil && upstreamData.Data.Inner != nil {
				errObj = upstreamData.Data.Inner.Error
			}
			util.DebugLog("上游错误: code=%d, detail=%s", errObj.Code, errObj.Detail)
			// 结束下游流
			endChunk := model.OpenAIResponse{
				ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   config.MODEL_NAME,
				Choices: []model.Choice{{Index: 0, Delta: model.Delta{}, FinishReason: "stop"}},
			}
			writeSSEChunk(w, endChunk)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		util.DebugLog("解析成功 - 类型: %s, 阶段: %s, 内容长度: %d, 完成: %v",
			upstreamData.Type, upstreamData.Data.Phase, len(upstreamData.Data.DeltaContent), upstreamData.Data.Done)

		// 策略2：总是展示thinking + answer
		if upstreamData.Data.DeltaContent != "" {
			var out = upstreamData.Data.DeltaContent
			if upstreamData.Data.Phase == "thinking" {
				out = transformThinking(out)
			}
			if out != "" {
				util.DebugLog("发送内容(%s): %s", upstreamData.Data.Phase, out)
				chunk := model.OpenAIResponse{
					ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   config.MODEL_NAME,
					Choices: []model.Choice{
						{
							Index: 0,
							Delta: model.Delta{Content: out},
						},
					},
				}
				writeSSEChunk(w, chunk)
				flusher.Flush()
			}
		}

		// 检查是否结束
		if upstreamData.Data.Done || upstreamData.Data.Phase == "done" {
			util.DebugLog("检测到流结束信号")
			// 发送结束chunk
			endChunk := model.OpenAIResponse{
				ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   config.MODEL_NAME,
				Choices: []model.Choice{
					{
						Index:        0,
						Delta:        model.Delta{},
						FinishReason: "stop",
					},
				},
			}
			writeSSEChunk(w, endChunk)
			flusher.Flush()

			// 发送[DONE]
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			util.DebugLog("流式响应完成，共处理%d行", lineCount)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		util.DebugLog("扫描器错误: %v", err)
	}
}

func writeSSEChunk(w http.ResponseWriter, chunk model.OpenAIResponse) {
	data, _ := json.Marshal(chunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func handleNonStreamResponseWithIDs(w http.ResponseWriter, upstreamReq model.UpstreamRequest, chatID string, authToken string) {
	util.DebugLog("开始处理非流式响应 (chat_id=%s)", chatID)

	resp, err := upstream.CallUpstreamWithHeaders(upstreamReq, chatID, authToken)
	if err != nil {
		util.DebugLog("调用上游失败: %v", err)
		http.Error(w, "Failed to call upstream", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		util.DebugLog("上游返回错误状态: %d", resp.StatusCode)
		// 读取错误响应体
		if config.DEBUG_MODE {
			body, _ := io.ReadAll(resp.Body)
			util.DebugLog("上游错误响应: %s", string(body))
		}
		http.Error(w, "Upstream error", http.StatusBadGateway)
		return
	}

	// 收集完整响应（策略2：thinking与answer都纳入，thinking转换）
	var fullContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	util.DebugLog("开始收集完整响应内容")

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "" {
			continue
		}

		var upstreamData model.UpstreamData
		if err := json.Unmarshal([]byte(dataStr), &upstreamData); err != nil {
			continue
		}

		if upstreamData.Data.DeltaContent != "" {
			out := upstreamData.Data.DeltaContent
			if upstreamData.Data.Phase == "thinking" {
				out = func(s string) string {
					// 同步一份转换逻辑（与流式一致）
					s = regexp.MustCompile(`(?s)<summary>.*?</summary>`).ReplaceAllString(s, "")
					s = strings.ReplaceAll(s, "</thinking>", "")
					s = strings.ReplaceAll(s, "<Full>", "")
					s = strings.ReplaceAll(s, "</Full>", "")
					s = strings.TrimSpace(s)
					switch config.THINK_TAGS_MODE {
					case "think":
						s = regexp.MustCompile(`<details[^>]*>`).ReplaceAllString(s, "<think>")
						s = strings.ReplaceAll(s, "</details>", "</think>")
					case "strip":
						s = regexp.MustCompile(`<details[^>]*>`).ReplaceAllString(s, "")
						s = strings.ReplaceAll(s, "</details>", "")
					}
					s = strings.TrimPrefix(s, "> ")
					s = strings.ReplaceAll(s, "\n> ", "\n")
					return strings.TrimSpace(s)
				}(out)
			}
			if out != "" {
				fullContent.WriteString(out)
			}
		}

		if upstreamData.Data.Done || upstreamData.Data.Phase == "done" {
			util.DebugLog("检测到完成信号，停止收集")
			break
		}
	}

	finalContent := fullContent.String()
	util.DebugLog("内容收集完成，最终长度: %d", len(finalContent))

	// 构造完整响应
	response := model.OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   config.MODEL_NAME,
		Choices: []model.Choice{
			{
				Index: 0,
				Message: model.Message{
					Role:    "assistant",
					Content: finalContent,
				},
				FinishReason: "stop",
			},
		},
		Usage: model.Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	util.DebugLog("非流式响应发送完成")
}
