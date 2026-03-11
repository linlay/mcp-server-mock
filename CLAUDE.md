# CLAUDE.md

## 项目概览

`mcp-server-mock` 是一个用于联调和回归测试的 Mock MCP Server，使用 Go `net/http` 提供单入口 `POST /mcp`。

当前状态：

- 开发中，可用于本地联调和协议回归
- 工具协议定义以 `tools/*.yml` 为唯一事实源
- 配置加载采用内嵌默认配置、可选外部 yml 和环境变量覆盖的分层模型

核心场景：

- 为 MCP 客户端或网关提供稳定的 mock 工具响应
- 在启动阶段校验工具定义与 handler 一致性
- 在调用阶段按 schema 校验 `tools/call.arguments`

## 技术栈

- 语言：Go `1.26`
- HTTP：标准库 `net/http`
- 配置与协议文件：`gopkg.in/yaml.v3`
- JSON Schema 校验：`github.com/santhosh-tekuri/jsonschema/v5`（通过本地 `third_party/jsonschema` 替换）
- 测试：Go 标准库 `testing`
- 运行形态：本地 `go run` / Docker Compose / Docker 镜像

## 架构设计

整体风格：单体 API 服务。

分层与职责：

- `cmd/mcp-server`
  - 组装配置、registry、controller，并启动 HTTP Server
- `internal/config`
  - 提供代码默认值、内嵌 `application.yml`、可选 `CONFIG_PATH` 外部 yml、环境变量覆盖的加载逻辑
- `internal/mcp/spec`
  - 读取 `tools/*.yml` 并校验顶层结构
- `internal/mcp/schema`
  - 编译并执行 `inputSchema` 校验
- `internal/mcp/tools`
  - 定义 handler 接口、注册表与各 mock 工具实现
- `internal/mcp/transport`
  - 处理 JSON-RPC 请求分发与 SSE 输出
- `internal/mcp/protocol`
  - 定义 RPC 请求、响应和错误码
- `internal/observability`
  - 负责日志、脱敏和请求体截断

关键设计决策：

- `tools/*.yml` 与 Go handler 必须双向一致，不一致时启动失败
- 内嵌默认配置不对运行时用户开放编辑；复杂结构化配置通过可选 `CONFIG_PATH` 外部 yml 追加
- 环境变量始终高于 yml，用于生产敏感项和最终覆盖

## 目录结构

- `cmd/`
  - 程序入口
- `internal/config/`
  - 配置结构、加载逻辑、配置测试、内嵌资源
- `internal/mcp/`
  - MCP 协议、tool spec、schema 校验、传输控制器与工具实现
- `internal/observability/`
  - 日志与脱敏
- `tools/`
  - mock 工具定义文件，作为工具元数据和 schema 的事实源
- `third_party/`
  - 本地替换的第三方依赖

目录组织原则：按技术层拆分，工具定义与实现分离维护。

## 数据结构

核心运行时结构：

- `config.Config`
  - `ServerPort`
  - `ToolsSpecLocationPattern`
  - `HTTPMaxBodyBytes`
  - `Observability`
- `config.ObservabilityConfig`
  - `LogEnabled`
  - `LogMaxBodyLength`
  - `LogIncludeHeaders`
- `spec.ToolSpec`
  - `Type`
  - `Name`
  - `Label`
  - `Description`
  - `AfterCallHint`
  - `InputSchema`

数据流：

1. 启动时加载配置
2. 根据配置读取 `tools/*.yml`
3. 编译每个工具的 `inputSchema`
4. 建立 tool spec 与 handler 的注册表
5. 请求进入 `/mcp` 后按 method 分发
6. `tools/call` 先校验参数，再执行 handler，并返回结构化结果

## API 定义

入口：

- `POST /mcp`

支持方法：

- `initialize`
- `tools/list`
- `tools/call`
- `viewports/list`
- `viewports/get`

`tools/list` 顶层扩展字段：

- `label`：工具的人类可读名称，可用于中文展示
- `toolAction: true`：action 工具
- `toolType` + `viewportKey`：仅在 `tools/*.yml` 显式声明时返回，服务端不做隐式推导
- 未声明扩展字段：backend 工具

`viewports/*` 约定：

- viewport 能力通过 MCP method 暴露，不提供独立 HTTP viewport 查询接口
- `viewports/list` 返回 `viewportKey`、`viewportType` 和 `toolNames`
- `viewports/get` 要求 `viewportKey`，返回 `viewportKey`、`viewportType` 和 `payload`
- `viewports/get` 缺少或传入不存在的 `viewportKey` 时返回 `-32602`

响应约定：

- 使用 JSON-RPC 2.0 风格请求/响应
- 当 `Accept: text/event-stream` 时支持 SSE 输出

错误码：

- `-32700`：parse error
- `-32600`：invalid request
- `-32601`：method not found
- `-32602`：invalid params
- `-32603`：internal error

兼容性约束：

- unknown tool 保持返回 `result.isError=true`，而不是 RPC 级 method not found

## 开发要点

- 新增工具时必须同时更新 `tools/<tool>.yml` 与 `BuiltinHandlers()`
- 参数规则以 YAML `inputSchema` 为准，不在 Go 代码中重复维护业务级参数校验
- 配置职责边界：
  - `internal/config/application.yml`：内嵌默认配置
  - `configs/*.yml`：可选外部结构化配置
  - `.env`：本地简单键值与可选 `CONFIG_PATH`
  - 环境变量：最终覆盖层
- Docker 镜像不应包含 `.env` 等本地私密文件
- README 只写使用、部署、排查；详细协议契约维护在 `docs/mcp-protocol-definition.md`
- 其余设计事实统一维护在本文件

## 开发流程

本仓库当前可见的开发流程事实：

- 本地运行：`make run`（会读取 `.env`，并优先使用 `HOST_PORT` 作为监听端口）
- 测试：`make test`
- Docker 本地编排：`make docker-up`
- 镜像构建：`make docker-build`

当前仓库中未看到可供引用的分支策略、提交规范或 CI 配置文件，因此这些流程在本文件中不扩展为事实。

## 已知约束与注意事项

- 当前项目配置项较少，默认不提供 `configs/config.dev.yml` 或 `configs/config.prod.yml`
- 若未来配置明显增多，可启用 `CONFIG_PATH` 和外部 yml，但外部 yml 只写需要覆写的字段
- `application.yml` 已内嵌进二进制；发布后不可将其作为外部配置文件修改
- 测试用例覆盖工具注册、HTTP 分发、SSE 和 observability，配置模块测试主要覆盖优先级与严格解析
