# mcp-server-mock

## 项目简介

`mcp-server-mock` 是一个用于联调和回归测试的 Mock MCP Server，使用 Go `net/http` 提供单入口 `POST /mcp`，支持 `initialize`、`tools/list`、`tools/call`、`viewports/list`、`viewports/get` 和可选 SSE 输出。

当前版本额外提供：

- 最小可用的 `bash` 工具，用于验证 Rena 到 MCP Server 的工具调用链
- `tools/call.params._meta` 扩展，用于透传工具调用上下文
- 仓库内维护的 MCP 协议与 viewport 协议文档

协议文档：

- [docs/mcp-protocol-definition.md](/Users/linlay-macmini/Project/mcp-server-mock/docs/mcp-protocol-definition.md)
- [docs/viewport-protocol-definition.md](/Users/linlay-macmini/Project/mcp-server-mock/docs/viewport-protocol-definition.md)

## 快速开始

环境要求：

- Go `1.26+`
- Docker 与 Docker Compose（容器方式运行时）

首次使用：

```bash
cp .env.example .env
```

本地运行：

```bash
make run
```

运行测试：

```bash
make test
```

Docker Compose 启动：

```bash
make docker-up
```

中国大陆网络环境可使用：

```bash
make docker-up-cn
```

默认访问地址：

```bash
http://localhost:11969/mcp
```

## 配置说明

配置层级：

1. 代码默认值
2. 内嵌默认配置 `internal/config/application.yml`
3. 可选外部 yml（设置 `CONFIG_PATH` 时启用）
4. 环境变量

常用变量：

- `HOST_PORT`：`make run` 与 Docker Compose 对宿主机暴露的端口
- `CONFIG_PATH`：可选外部结构化配置文件
- `SERVER_PORT`：服务监听端口
- `MCP_TOOLS_SPEC_LOCATION_PATTERN`：tool spec 文件匹配路径
- `MCP_VIEWPORTS_DIR`：viewport 文件目录
- `MCP_HTTP_MAX_BODY_BYTES`：HTTP 请求体大小上限
- `MCP_OBSERVABILITY_LOG_ENABLED`：是否开启 observability 日志
- `MCP_OBSERVABILITY_LOG_MAX_BODY_LENGTH`：日志摘要截断长度
- `MCP_OBSERVABILITY_LOG_INCLUDE_HEADERS`：是否记录请求头
- `MCP_BASH_WORKING_DIRECTORY`：bash 默认工作目录
- `MCP_BASH_ALLOWED_ROOTS`：bash 允许访问的根目录列表，逗号分隔
- `MCP_BASH_ALLOWED_COMMANDS`：bash 允许执行的命令白名单，逗号分隔
- `MCP_BASH_TIMEOUT_MS`：bash 单次执行超时
- `MCP_BASH_MAX_COMMAND_CHARS`：bash 命令最大长度
- `MCP_BASH_MAX_OUTPUT_CHARS`：bash 输出最大长度

说明：

- `.env` 仅用于本地，不提交到仓库
- `.env.example` 是环境变量契约
- 外部 yml 只写覆盖字段，不复制全部默认值

## 部署

直接运行二进制：

```bash
make build
./mcp-server
```

构建镜像：

```bash
make docker-build
```

运行镜像：

```bash
docker run --rm -p 8080:8080 --env-file .env mcp-server-mock
```

部署约束：

- 镜像内仅额外安装 `bash`，不再增加 `git`、`rg` 等工具
- 不将 `.env` 打包进镜像
- 不把真实密钥写入 `Dockerfile`
- 生产敏感项通过环境变量或 Secret 注入

## 运维

基础探活：

- 向 `/mcp` 发送 `initialize`、`tools/list` 或 `viewports/list` 请求，确认返回 `200`

示例：

```bash
curl -sS -X POST "http://localhost:11969/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "tools/list",
    "params": {}
  }'
```

`bash` + `_meta` 示例：

```bash
curl -sS -X POST "http://localhost:11969/mcp" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": "2",
    "method": "tools/call",
    "params": {
      "name": "bash",
      "arguments": {
        "command": "pwd"
      },
      "_meta": {
        "workDirectory": "./viewports",
        "userId": "rena-user-1"
      }
    }
  }'
```

常见排查：

- 启动失败并提示 registry 错误：检查 `tools/*.yml` 与 `BuiltinHandlers()` 是否一一对应
- 启动失败并提示 `read viewports dir`：检查镜像/运行目录中是否包含 `viewports/`
- `tools/call` 返回 `-32602`：检查 `arguments` 是否满足对应 tool 的 `inputSchema`
- `bash` 返回 `exitCode=-1`：检查命令是否在白名单内，或 `_meta.workDirectory` / 路径参数是否越出允许根目录
- Docker 构建阶段 `go mod download` 超时：可改用 `make docker-build-cn` 或 `make docker-up-cn`
