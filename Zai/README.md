# 🛡️ Zai OpenAI Proxy

[![Go Version](https://img.shields.io/badge/Go-1.23.3-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Made with Love](https://img.shields.io/badge/Made%20with-%E2%9D%A4%EF%B8%8F-red.svg)]()

这是一个将 [Z.ai](https://chat.z.ai/) 的聊天 API 转换为 OpenAI 兼容格式的 Go 语言代理服务。通过本项目，您可以将任何支持 OpenAI API 的客户端无缝对接到 Z.ai 的模型上。

## ✨ 功能特性

- **OpenAI 兼容**: 完全兼容 OpenAI 的 `/v1/chat/completions` 和 `/v1/models` 接口。
- **流式与非流式**: 同时支持流式（Server-Sent Events）和非流式响应。
- **简易鉴权**: 通过可配置的 `Bearer Token` 进行服务认证。
- **动态 Token**: 支持自动获取 Z.ai 的匿名 `token`，避免多客户端共享记忆。
- **高度可配**: 核心参数均可通过 `internal/config/config.go` 文件进行配置。
- **跨域支持**: 内置 CORS 配置，方便前端应用直接调用。
- **调试模式**: 可开启 Debug 模式以输出详细的请求和响应日志。

## 🚀 快速开始

### 环境要求

- [Go](https://golang.org/dl/) (版本 >= 1.23)

### 安装与运行

1.  **克隆或下载项目**
    ```bash
    # 假设您已将项目代码放置在 Zai 目录中
    ```

2.  **整理依赖**
    进入项目根目录，执行以下命令来下载和整理模块依赖：
    ```bash
    go mod tidy
    ```

3.  **运行服务**
    ```bash
    go run .
    ```

4.  **服务启动**
    当您在终端看到以下输出时，表示服务已成功启动：
    ```
    OpenAI兼容API服务器启动在端口:8080
    模型: GLM-4.5
    上游: https://chat.z.ai/api/chat/completions
    Debug模式: true
    ```

## ⚙️ 配置说明

所有配置项均位于 `internal/config/config.go` 文件中。

| 常量名 | 类型 | 描述 |
| :--- | :--- | :--- |
| `UPSTREAM_URL` | `string` | Z.ai 的上游 API 地址。 |
| `DEFAULT_KEY` | `string` | **下游鉴权密钥**。客户端在请求时 `Authorization` 头中需要携带的 `Bearer Token`。 |
| `UPSTREAM_TOKEN` | `string` | **上游 Z.ai 备用 Token**。当自动获取匿名 Token 失败时，会使用此 Token。您可以从 Z.ai 网站的请求中获取自己的 Token 填入。 |
| `MODEL_NAME` | `string` | 在 `/v1/models` 接口中向客户端展示的模型名称。 |
| `PORT` | `string` | 服务监听的端口号。 |
| `DEBUG_MODE` | `bool` | 是否开启调试模式。开启后会打印详细日志。 |
| `ANON_TOKEN_ENABLED` | `bool` | 是否启用自动获取 Z.ai 匿名 Token 的功能。 |

## 🎮 使用方法

服务启动后，您可以通过向 `http://localhost:8080/v1/chat/completions` 发送 `POST` 请求来使用。

请确保在请求的 `Header` 中加入了正确的鉴权信息。

### cURL 示例

以下是一个使用 `cURL` 请求的例子：

```bash
curl --location 'http://localhost:8080/v1/chat/completions' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer sk-your-key' \
--data '{
    "model": "GLM-4.5",
    "messages": [
        {
            "role": "user",
            "content": "你好，请介绍一下你自己"
        }
    ],
    "stream": false
}'
```

**参数说明:**
- **`Authorization: Bearer sk-your-key`**: 这里的 `sk-your-key` 必须与 `config.go` 文件中的 `DEFAULT_KEY` 值保持一致。
- **`stream: false`**: 如果需要使用流式响应，请将其设置为 `true`。

---
*Enjoy!*
