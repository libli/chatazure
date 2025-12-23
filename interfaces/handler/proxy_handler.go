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
	isAuthenticated, username := p.authRequest(w, r)
	if !isAuthenticated {
		return
	}

	body, _ := io.ReadAll(r.Body)

	// 从 body 中获取模型名称
	var result map[string]interface{}
	_ = json.Unmarshal(body, &result)
	modelValue, _ := result["model"].(string)

	// 将模型名称从请求中映射到部署名称
	deploymentName, exists := p.azureConfig.Deployments[modelValue]
	if !exists {
		http.Error(w, "Unsupported model", http.StatusBadRequest)
		tlog.Info.Printf("<<%s>> request unsupported model: %s", username, modelValue)
		return
	}

	// Restore the io.ReadCloser to its original state
	// 如果没有这一步，r.Body 会被读取后就为空
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// 更新调用次数
	p.userLogic.UpdateCount(username)

	director := func(req *http.Request) {
		originURL := req.URL.String()
		req = p.setupAzureRequest(req, "/openai/deployments/"+deploymentName+"/chat/completions")
		tlog.Info.Printf("<<%s>> request [%s] proxying: %s -> %s", username, modelValue, originURL, req.URL.String())
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

// HandleResponses is the handler for /v1/responses path.
// It keeps the OpenAI-compatible request shape and rewrites `model` to Azure deployment name.
func (p *ProxyHandler) HandleResponses(w http.ResponseWriter, r *http.Request) {
	handleCORSRequest(w, r)
	isAuthenticated, username := p.authRequest(w, r)
	if !isAuthenticated {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	modelValue, _ := payload["model"].(string)
	if modelValue == "" {
		http.Error(w, "Missing model", http.StatusBadRequest)
		return
	}

	// Map OpenAI model name -> Azure deployment name.
	deploymentName, exists := p.azureConfig.Deployments[modelValue]
	if !exists {
		http.Error(w, "Unsupported model", http.StatusBadRequest)
		tlog.Info.Printf("<<%s>> request unsupported model (responses): %s", username, modelValue)
		return
	}
	payload["model"] = deploymentName

	newBody, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to encode request", http.StatusInternalServerError)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(newBody))
	r.ContentLength = int64(len(newBody))

	// 更新调用次数
	p.userLogic.UpdateCount(username)

	director := func(req *http.Request) {
		originURL := req.URL.String()
		req = p.setupAzureRequest(req, p.getAzureResponsesPath())
		tlog.Info.Printf("<<%s>> request [responses:%s] proxying: %s -> %s", username, modelValue, originURL, req.URL.String())
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

// HandleAzureResponsesPassthrough proxies Azure native responses endpoints under /openai/**,
// but only allows responses-related paths (whitelist).
func (p *ProxyHandler) HandleAzureResponsesPassthrough(w http.ResponseWriter, r *http.Request) {
	handleCORSRequest(w, r)
	isAuthenticated, username := p.authRequest(w, r)
	if !isAuthenticated {
		return
	}

	if !isAllowedAzureResponsesPath(r.URL.Path) {
		http.NotFound(w, r)
		tlog.Warn.Printf("<<%s>> blocked azure passthrough path: %s", username, r.URL.Path)
		return
	}

	switch r.Method {
	case http.MethodPost, http.MethodGet, http.MethodDelete:
		// allowed
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Best-effort model rewrite for POST bodies: if `model` matches OpenAI alias, rewrite to deployment.
	if r.Method == http.MethodPost && r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			var payload map[string]interface{}
			if json.Unmarshal(body, &payload) == nil {
				if modelValue, ok := payload["model"].(string); ok && modelValue != "" {
					if deploymentName, exists := p.azureConfig.Deployments[modelValue]; exists {
						payload["model"] = deploymentName
						if newBody, err := json.Marshal(payload); err == nil {
							body = newBody
						}
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
		path := req.URL.Path
		req = p.setupAzureRequest(req, path)
		tlog.Info.Printf("<<%s>> request [azure-responses] proxying: %s -> %s", username, originURL, req.URL.String())
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
	isAuthenticated, username, _ := p.userLogic.Auth(token)
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
	// Use Set to avoid duplicating api-version if caller already provided one.
	query.Set("api-version", p.azureConfig.ApiVersion)
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

func (p *ProxyHandler) getAzureResponsesPath() string {
	if strings.TrimSpace(p.azureConfig.ResponsesPath) != "" {
		return p.azureConfig.ResponsesPath
	}
	// Default to v1 path (recommended by the docs); can be overridden by config.
	return "/openai/v1/responses"
}

func isAllowedAzureResponsesPath(path string) bool {
	if path == "/openai/responses" || path == "/openai/v1/responses" {
		return true
	}
	if strings.HasPrefix(path, "/openai/responses/") || strings.HasPrefix(path, "/openai/v1/responses/") {
		return true
	}
	return false
}
