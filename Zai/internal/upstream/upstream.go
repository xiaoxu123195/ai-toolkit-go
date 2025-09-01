package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"Zai/internal/config"
	"Zai/internal/model"
	"Zai/internal/util"
)

// GetAnonymousToken 获取匿名token（每次对话使用不同token，避免共享记忆）
func GetAnonymousToken() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", config.ORIGIN_BASE+"/api/v1/auths/", nil)
	if err != nil {
		return "", err
	}
	// 伪装浏览器头
	req.Header.Set("User-Agent", config.BROWSER_UA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("X-FE-Version", config.X_FE_VERSION)
	req.Header.Set("sec-ch-ua", config.SEC_CH_UA)
	req.Header.Set("sec-ch-ua-mobile", config.SEC_CH_UA_MOB)
	req.Header.Set("sec-ch-ua-platform", config.SEC_CH_UA_PLAT)
	req.Header.Set("Origin", config.ORIGIN_BASE)
	req.Header.Set("Referer", config.ORIGIN_BASE+"/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anon token status=%d", resp.StatusCode)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Token == "" {
		return "", fmt.Errorf("anon token empty")
	}
	return body.Token, nil
}

func CallUpstreamWithHeaders(upstreamReq model.UpstreamRequest, refererChatID string, authToken string) (*http.Response, error) {
	reqBody, err := json.Marshal(upstreamReq)
	if err != nil {
		util.DebugLog("上游请求序列化失败: %v", err)
		return nil, err
	}

	util.DebugLog("调用上游API: %s", config.UPSTREAM_URL)
	util.DebugLog("上游请求体: %s", string(reqBody))

	req, err := http.NewRequest("POST", config.UPSTREAM_URL, bytes.NewBuffer(reqBody))
	if err != nil {
		util.DebugLog("创建HTTP请求失败: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("User-Agent", config.BROWSER_UA)
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept-Language", "zh-CN")
	req.Header.Set("sec-ch-ua", config.SEC_CH_UA)
	req.Header.Set("sec-ch-ua-mobile", config.SEC_CH_UA_MOB)
	req.Header.Set("sec-ch-ua-platform", config.SEC_CH_UA_PLAT)
	req.Header.Set("X-FE-Version", config.X_FE_VERSION)
	req.Header.Set("Origin", config.ORIGIN_BASE)
	req.Header.Set("Referer", config.ORIGIN_BASE+"/c/"+refererChatID)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		util.DebugLog("上游请求失败: %v", err)
		return nil, err
	}

	util.DebugLog("上游响应状态: %d %s", resp.StatusCode, resp.Status)
	return resp, nil
}
