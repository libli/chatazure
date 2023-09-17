# azure openai 代理

## 特性

1. 后端使用 Azure OpenAI 服务，适配所有 OpenAI 客户端
2. 支持用户管理功能，为每个用户分配独立的 key，团队使用
3. 统计每个用户的 API 调用次数
4. 支持 SSE

## 部署

1. 创建配置文件 config.yaml，参考 config.yaml.example

users 是你想分配的用户名和密码。用户用该配置文件里的 password 即可在各种 OpenAI 客户端中调用 Azure 的 API。

2. 创建一个空的数据库文件 chat.db:

```bash
touch chat.db
```

3. 运行 docker:

```bash
docker run --name=chatazure -d \
  --restart=unless-stopped -p 8080:8080 \
  -v /data/chatazure/config.yaml:/app/config.yaml \
  -v /data/chatazure/chat.db:/app/chat.db \
  libli/chatazure:latest
```

把上面命令中的 `/data/chatazure` 替换为你的配置文件和数据库文件所在的目录。

4. 后续可用 sqlite3 管理数据库，确保已经安装 sqlite3 客户端，例如：

```bash
sqlite3 chat.db
sqlite> select * from users;
sqlite> insert into users (username, token) VALUES ('***', '****');
```

4. 如果需要支持 https 协议，使用 nginx 反向代理即可。参考如下配置（支持 SSE）：

```nginx
server {
    listen       80;
    listen       [::]:80;
    server_name  api.exapmle.com;
    return       301 https://$host$request_uri;
}

# Settings https.
server {
    listen       443 ssl http2;
    listen       [::]:443 ssl http2;
    server_name  api.exapmle.com;

    ssl_certificate             "/etc/pki/nginx/api.exapmle.com.crt";
    ssl_certificate_key         "/etc/pki/nginx/private/api.exapmle.com.key";
    ssl_session_cache           shared:SSL:1m;
    ssl_session_timeout         10m;
    ssl_ciphers                 HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers   on;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location / {
        proxy_pass       http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        chunked_transfer_encoding off;
        proxy_buffering off;
    }
}
```

## 一些用法

### OpenCat 客户端 (iOS Mac)

OpenCat 虽然有了团队版，但是划在收费版功能。可以直接利用自定义 OpenAI 域名的功能达到团队免费使用。

1. 在自己的服务器部署本服务，OpenAI Key 只在自己服务器上保存，不会泄露。
2. 在 OpenCat 客户端中，设置 API key 为 config.yaml 中自己设置的 Token。
3. 在 OpenCat 客户端中，设置自定义 OpenAI 域名为自己的服务器地址，例如：https://api.exapmle.com 点击“自定义 API 域名”下的“验证”即可。

### ChatBox (Windows, Mac, Linux)

https://github.com/Bin-Huang/chatbox

在设置中填入 API 域名为本服务部署的地址，API key 为自己在 sqlite 中分配的密钥。

目前 chatbox 只支持 https 的 API 域名，所以需要在 nginx 中配置 https。

### ChatBoost（Android）

设置自定义 API 地址为本服务部署的地址，客户端上的 OpenAI API 密钥设为自己在 sqlite 中分配的密钥。

### ChatGPT Box（浏览器插件）

https://chrome.google.com/webstore/detail/chatgptbox/eobbhoofkanlmddnplfhnmkfbnlhpbbo

在“高级”中的“自定义的 ChatGPT 网页 API 地址”修改为部署本 docker 的域名，“API 模式”后面的框中输入本 Docker 中自己在 sqlite3 中插入的 key。

### ChatGPT-Next-web（网页版）

https://github.com/Yidadaa/ChatGPT-Next-Web

部署该 docker 时参考如下：

```
docker run --name=chatgpt -d --restart=unless-stopped \
  -p 3000:3000 \
  -e OPENAI_API_KEY="" \
  -e BASE_URL="http://myapi.com" \
  yidadaa/chatgpt-next-web:latest
```

把 BASE_URL 替换为部署本 docker 的服务器域名，OPENAI_API_KEY 留空，不需要填。
这样用户访问网页版时，直接使用自己分配的 KEY 即可。

### xcatliu/chatgpt-next (网页版)

https://github.com/xcatliu/chatgpt-next

部署该 docker 时参考如下：

```
docker run --name=chatgpt-next -d --restart=unless-stopped \
  -p 3000:3000 \
  -e CHATGPT_NEXT_API_HOST="http://api.example.com:8080" \
  xcatliu/chatgpt-next:latest
```

## 开源协议

MIT，随便拿去用，记得多帮我宣传宣传。

如果觉得帮助到你了，欢迎请[我喝一杯咖啡](https://github.com/libli/buy-me-a-coffee) ☕️。
