package main

import (
	"log"
	"net/http"

	"Zai/internal/config"
	"Zai/internal/handler"
)

func main() {
	http.HandleFunc("/v1/models", handler.HandleModels)
	http.HandleFunc("/v1/chat/completions", handler.HandleChatCompletions)
	http.HandleFunc("/", handler.HandleOptions)

	log.Printf("OpenAI兼容API服务器启动在端口%s", config.PORT)
	log.Printf("模型: %s", config.MODEL_NAME)
	log.Printf("上游: %s", config.UPSTREAM_URL)
	log.Printf("Debug模式: %v", config.DEBUG_MODE)
	log.Fatal(http.ListenAndServe(config.PORT, nil))
}
