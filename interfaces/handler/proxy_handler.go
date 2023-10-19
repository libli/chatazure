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

// HandleChat is the handler for /v1/chat/completions path.
func (p *ProxyHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	handleCORSRequest(w, r)
	isAuthenticated, username, canUseGPT4 := p.authRequest(w, r)
	if !isAuthenticated {
		return
	}

	shouldProxy := true
	director := func(req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		// Restore the io.ReadCloser to its original state
		// 如果没有这一步，req.Body 会被读取后就为空
		req.Body = io.NopCloser(bytes.NewBuffer(body))

		// 从 body 中获取模型名称
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		modelValue, _ := result["model"].(string)

		// 将模型名称从请求中映射到部署名称
		deploymentName, exists := p.azureConfig.Deployments[modelValue]
		if !exists {
			http.Error(w, "Unsupported model", http.StatusBadRequest)
			shouldProxy = false
			req.URL = nil // 必须设置为 nil，否则会继续执行 proxy
			return
		}

		// 如果用户没有权限使用 gpt-4，且请求的模型是 gpt-4，则返回 403
		if strings.HasPrefix(modelValue, "gpt-4") && !canUseGPT4 {
			http.Error(w, "Forbidden Use GPT-4", http.StatusForbidden)
			shouldProxy = false
			req.URL = nil
			return
		}

		// 更新调用次数
		p.userLogic.UpdateCount(username)

		originURL := req.URL.String()
		req = p.setupAzureRequest(req, "/openai/deployments/"+deploymentName+"/chat/completions")

		tlog.Info.Printf("<<%s>> request [%s] proxying: %s -> %s", username, modelValue, originURL, req.URL.String())
	}

	if shouldProxy {
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(w, r)
	}
}

// HandleModels is the handler for /v1/models path.
func (p *ProxyHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	handleCORSRequest(w, r)
	isAuthenticated, username, _ := p.authRequest(w, r)
	if !isAuthenticated {
		return
	}

	// https://learn.microsoft.com/en-us/rest/api/cognitiveservices/azureopenaistable/models/list?tabs=HTTP
	director := func(req *http.Request) {
		originURL := req.URL.String()
		req = p.setupAzureRequest(req, "/openai/models")

		tlog.Info.Printf("<<%s>> request proxying: %s -> %s", username, originURL, req.URL.String())
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

// authRequest authenticates the request.
func (p *ProxyHandler) authRequest(w http.ResponseWriter, r *http.Request) (bool, string, bool) {
	token := extractAuthToken(r)
	isAuthenticated, username, canUseGPT4 := p.userLogic.Auth(token)
	if !isAuthenticated {
		tlog.Warn.Printf("unauthorized request: %s", token)
		response.Unauthorized(w)
		return false, "", false
	}
	tlog.Info.Printf("authorized user: %s", username)
	return true, username, canUseGPT4
}

// setupAzureRequest sets up the Azure request.
func (p *ProxyHandler) setupAzureRequest(req *http.Request, path string) *http.Request {
	req = p.setupAzureHeader(req)
	req = p.setupAzureEndpoint(req)
	req.URL.Path = path
	req.URL.RawPath = req.URL.EscapedPath()
	query := req.URL.Query()
	query.Add("api-version", p.azureConfig.ApiVersion)
	req.URL.RawQuery = query.Encode()
	return req
}

// setupAzureHeader sets up the proxy request header.
func (p *ProxyHandler) setupAzureHeader(req *http.Request) *http.Request {
	req.Header.Set("api-key", p.azureConfig.ApiKey)
	req.Header.Del("Authorization")
	return req
}

// setupAzureEndpoint sets up the Azure endpoint.
func (p *ProxyHandler) setupAzureEndpoint(req *http.Request) *http.Request {
	parseEndpoint, _ := url.Parse(p.azureConfig.Endpoint)
	req.Host = parseEndpoint.Host
	req.URL.Scheme = parseEndpoint.Scheme
	req.URL.Host = parseEndpoint.Host
	return req
}
