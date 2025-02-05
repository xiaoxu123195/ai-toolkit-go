package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OpenAIRequest struct {
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Model    string    `json:"model"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MerlinRequest struct {
	Attachments []interface{} `json:"attachments"`
	ChatId      string        `json:"chatId"`
	Language    string        `json:"language"`
	Message     struct {
		Content  string `json:"content"`
		Context  string `json:"context"`
		ChildId  string `json:"childId"`
		Id       string `json:"id"`
		ParentId string `json:"parentId"`
	} `json:"message"`
	Metadata struct {
		LargeContext  bool `json:"largeContext"`
		MerlinMagic   bool `json:"merlinMagic"`
		ProFinderMode bool `json:"proFinderMode"`
		WebAccess     bool `json:"webAccess"`
	} `json:"metadata"`
	Mode  string `json:"mode"`
	Model string `json:"model"`
}

type MerlinResponse struct {
	Data struct {
		Content string `json:"content"`
	} `json:"data"`
}

type OpenAIResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type TokenResponse struct {
	IdToken string `json:"idToken"`
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getToken() (string, error) {
	tokenReq := struct {
		UUID string `json:"uuid"`
	}{
		UUID: getEnvOrDefault("UUID", ""),
	}

	tokenReqBody, _ := json.Marshal(tokenReq)
	resp, err := http.Post(
		"https://getmerlin-main-server.vercel.app/generate",
		"application/json",
		strings.NewReader(string(tokenReqBody)),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.IdToken, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	authToken := r.Header.Get("Authorization")
	envToken := getEnvOrDefault("AUTH_TOKEN", "")

	if envToken != "" && authToken != "Bearer "+envToken {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.URL.Path != "/v1/chat/completions" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"GetMerlin2Api Service Running...","message":"MoLoveSze..."}`)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var openAIReq OpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&openAIReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var contextMessages []string
	for i := 0; i < len(openAIReq.Messages)-1; i++ {
		msg := openAIReq.Messages[i]
		contextMessages = append(contextMessages, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
	}
	context := strings.Join(contextMessages, "\n")
	merlinReq := MerlinRequest{
		Attachments: make([]interface{}, 0),
		ChatId:      generateV1UUID(),
		Language:    "AUTO",
		Message: struct {
			Content  string `json:"content"`
			Context  string `json:"context"`
			ChildId  string `json:"childId"`
			Id       string `json:"id"`
			ParentId string `json:"parentId"`
		}{
			Content:  openAIReq.Messages[len(openAIReq.Messages)-1].Content,
			Context:  context,
			ChildId:  generateUUID(),
			Id:       generateUUID(),
			ParentId: "root",
		},
		Mode:  "UNIFIED_CHAT",
		Model: openAIReq.Model,
		Metadata: struct {
			LargeContext  bool `json:"largeContext"`
			MerlinMagic   bool `json:"merlinMagic"`
			ProFinderMode bool `json:"proFinderMode"`
			WebAccess     bool `json:"webAccess"`
		}{
			LargeContext:  false,
			MerlinMagic:   false,
			ProFinderMode: false,
			WebAccess:     false,
		},
	}
	token, err := getToken()
	if err != nil {
		http.Error(w, "Failed to get token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	client := &http.Client{}
	merlinReqBody, _ := json.Marshal(merlinReq)

	req, _ := http.NewRequest("POST", "https://arcane.getmerlin.in/v1/thread/unified", strings.NewReader(string(merlinReqBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-merlin-version", "web-merlin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("sec-ch-ua", `"Not(A:Brand";v="99", "Microsoft Edge";v="133", "Chromium";v="133"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "Windows")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("host", "arcane.getmerlin.in")
	if openAIReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Transfer-Encoding", "chunked")
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if !openAIReq.Stream {
		var fullContent string
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				continue
			}

			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "event: message") {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					continue
				}
				dataLine = strings.TrimSpace(dataLine)

				if strings.HasPrefix(dataLine, "data: ") {
					dataStr := strings.TrimPrefix(dataLine, "data: ")
					var merlinResp MerlinResponse
					if err := json.Unmarshal([]byte(dataStr), &merlinResp); err != nil {
						continue
					}
					if merlinResp.Data.Content != " " {
						fullContent += merlinResp.Data.Content
					}
				}
			}
		}

		response := map[string]interface{}{
			"id":      generateUUID(),
			"object":  "chat.completion",
			"created": getCurrentTimestamp(),
			"model":   openAIReq.Model,
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": fullContent,
					},
					"finish_reason": "stop",
					"index":         0,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if strings.HasPrefix(line, "event: message") {
			dataLine, _ := reader.ReadString('\n')
			var merlinResp MerlinResponse
			json.Unmarshal([]byte(strings.TrimPrefix(dataLine, "data: ")), &merlinResp)

			if merlinResp.Data.Content != "" {
				openAIResp := OpenAIResponse{
					Id:      generateUUID(),
					Object:  "chat.completion.chunk",
					Created: getCurrentTimestamp(),
					Model:   openAIReq.Model,
					Choices: []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
						Index        int    `json:"index"`
						FinishReason string `json:"finish_reason"`
					}{{
						Delta: struct {
							Content string `json:"content"`
						}{
							Content: merlinResp.Data.Content,
						},
						Index:        0,
						FinishReason: "",
					}},
				}

				respData, _ := json.Marshal(openAIResp)
				fmt.Fprintf(w, "data: %s\n\n", string(respData))
			}
		}
	}

	finalResp := OpenAIResponse{
		Choices: []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
		}{{
			Delta: struct {
				Content string `json:"content"`
			}{Content: ""},
			Index:        0,
			FinishReason: "stop",
		}},
	}
	respData, _ := json.Marshal(finalResp)
	fmt.Fprintf(w, "data: %s\n\n", string(respData))
	fmt.Fprintf(w, "data: [DONE]\n\n")
}

func generateUUID() string {
	return uuid.New().String()
}

func generateV1UUID() string {
	uuidObj := uuid.Must(uuid.NewUUID())
	return uuidObj.String()
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
