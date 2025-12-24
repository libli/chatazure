package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"chatazure/config"
	"chatazure/interfaces/response"
	"chatazure/logic"
	"chatazure/repo"
	"chatazure/tlog"
)

// ProxyHandler is the handler for the openai proxy.
type ProxyHandler struct {
	userLogic   *logic.UserLogic
	azureConfig config.AzureConfig
}

// NewProxyHandler creates a new ProxyHandler.
func NewProxyHandler(user *repo.UserRepo, config config.AzureConfig) *ProxyHandler {
	return &ProxyHandler{
		userLogic:   logic.NewUserLogic(user),
		azureConfig: config,
	}
}

// HandleProxy is the unified handler for all OpenAI-compatible API requests.
// It adds /openai prefix to /v1/* paths and rewrites model names to Azure deployment names.
func (p *ProxyHandler) HandleProxy(w http.ResponseWriter, r *http.Request) {
	handleCORSRequest(w, r)
	isAuthenticated, username := p.authRequest(w, r)
	if !isAuthenticated {
		return
	}

	// 对于有 body 的请求，尝试映射 model 名称
	if r.Method == http.MethodPost && r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err == nil && len(body) > 0 {
			var payload map[string]interface{}
			if json.Unmarshal(body, &payload) == nil {
				if modelValue, ok := payload["model"].(string); ok && modelValue != "" {
					// 检查是否需要启用 web_search
					if p.shouldEnableWebSearch(modelValue) {
						payload["tools"] = []map[string]interface{}{
							{"type": "web_search_preview"},
						}
						tlog.Info.Printf("<<%s>> enabled web_search for model: %s", username, modelValue)
					}
					// 如果有映射配置，则替换 model 名称
					if deploymentName, exists := p.azureConfig.Deployments[modelValue]; exists {
						payload["model"] = deploymentName
						tlog.Info.Printf("<<%s>> model mapping: %s -> %s", username, modelValue, deploymentName)
					} else {
						// 没有映射配置，使用原始 model
						tlog.Info.Printf("<<%s>> model: %s (no mapping)", username, modelValue)
					}
					// 重新序列化 body
					if newBody, err := json.Marshal(payload); err == nil {
						body = newBody
					}
				}
			}
			r.Body = io.NopCloser(bytes.NewBuffer(body))
			r.ContentLength = int64(len(body))
		}
	}

	// 更新调用次数
	p.userLogic.UpdateCount(username)

	director := func(req *http.Request) {
		originURL := req.URL.String()
		originPath := req.URL.Path

		// 计算目标路径：/v1/* -> /openai/v1/*，/openai/* 保持不变
		targetPath := originPath
		if strings.HasPrefix(originPath, "/v1/") {
			targetPath = "/openai" + originPath
		}

		req = p.setupAzureRequest(req, targetPath)
		tlog.Info.Printf("<<%s>> proxying: %s -> %s", username, originURL, req.URL.String())
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

// authRequest authenticates the request.
func (p *ProxyHandler) authRequest(w http.ResponseWriter, r *http.Request) (bool, string) {
	token := extractAuthToken(r)
	isAuthenticated, username := p.userLogic.Auth(token)
	if !isAuthenticated {
		tlog.Warn.Printf("unauthorized request: %s", token)
		response.Unauthorized(w)
		return false, ""
	}
	tlog.Info.Printf("authorized user: %s", username)
	return true, username
}

// setupAzureRequest sets up the Azure request (adds prefix, replaces auth header).
func (p *ProxyHandler) setupAzureRequest(req *http.Request, path string) *http.Request {
	// 替换认证 header
	req.Header.Set("api-key", p.azureConfig.ApiKey)
	req.Header.Del("Authorization")

	// 设置 Azure endpoint
	parseEndpoint, _ := url.Parse(p.azureConfig.Endpoint)
	req.Host = parseEndpoint.Host
	req.URL.Scheme = parseEndpoint.Scheme
	req.URL.Host = parseEndpoint.Host
	req.URL.Path = path
	req.URL.RawPath = req.URL.EscapedPath()

	return req
}

// shouldEnableWebSearch 检查模型是否需要启用 web_search 功能
func (p *ProxyHandler) shouldEnableWebSearch(model string) bool {
	for _, m := range p.azureConfig.WebSearchModels {
		if m == model {
			return true
		}
	}
	return false
}
