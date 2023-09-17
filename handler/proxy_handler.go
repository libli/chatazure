package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"chatazure/config"
	"chatazure/repo"
	"chatazure/tlog"
)

// ProxyHandler is the handler for the openai proxy.
type ProxyHandler struct {
	user        *repo.UserRepo
	azureConfig config.AzureConfig
}

// NewProxyHandler creates a new ProxyHandler.
func NewProxyHandler(user *repo.UserRepo, config config.AzureConfig) *ProxyHandler {
	return &ProxyHandler{
		user:        user,
		azureConfig: config,
	}
}

// Proxy is the handler for the openai proxy.
func (p *ProxyHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	// CORS
	if r.Method == http.MethodOptions {
		handleOPTIONS(w)
		return
	}

	auth := r.Header.Get("Authorization")

	tlog.Info.Printf("auth: %s", auth)
	if auth == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}
	password := strings.TrimPrefix(auth, "Bearer ")
	if !p.checkUser(password) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
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

		req.Header.Set("api-key", p.azureConfig.ApiKey)
		req.Header.Del("Authorization")
		originURL := req.URL.String()
		req = p.convertReq(req, deploymentName)
		tlog.Info.Printf("proxying request [%s] %s -> %s", modelValue, originURL, req.URL.String())
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(w, r)
}

// checkUser checks if the password is valid.
func (p *ProxyHandler) checkUser(password string) bool {
	user := p.user.GetByPassword(password)
	if user == nil || user.Password != password {
		tlog.Info.Printf("user password: %s not found", password)
		return false
	}
	tlog.Info.Printf("user: %s invoke", user.Username)
	p.user.UpdateCount(user)
	return true
}

// convertReq converts the request to the Azure API.
func (p *ProxyHandler) convertReq(r *http.Request, deploymentName string) *http.Request {
	parseEndpoint, _ := url.Parse(p.azureConfig.Endpoint)
	r.Host = parseEndpoint.Host
	r.URL.Scheme = parseEndpoint.Scheme
	r.URL.Host = parseEndpoint.Host

	// Remove the api version from the path
	apiBase := "/v1"
	r.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deploymentName), strings.Replace(r.URL.Path, apiBase+"/", "/", 1))
	r.URL.RawPath = r.URL.EscapedPath()
	query := r.URL.Query()
	query.Add("api-version", p.azureConfig.ApiVersion)
	r.URL.RawQuery = query.Encode()
	return r
}

// handleOPTIONS handles the OPTIONS request.
func handleOPTIONS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.WriteHeader(http.StatusOK)
}
