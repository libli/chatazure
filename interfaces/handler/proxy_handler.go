package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

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
	isAuthenticated, username := p.authRequest(w, r)
	if !isAuthenticated {
		return
	}

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
			return
		}

		originURL := req.URL.String()
		req = p.setupAzureRequest(req, "/openai/deployments/"+deploymentName+"/chat/completions")

		tlog.Info.Printf("<<%s>> request [%s] proxying: %s -> %s", username, modelValue, originURL, req.URL.String())
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

// HandleModels is the handler for /v1/models path.
func (p *ProxyHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	handleCORSRequest(w, r)
	isAuthenticated, username := p.authRequest(w, r)
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
