# CLAUDE.md

## 项目概览

`mcp-server-mock` 是一个用于联调和回归测试的 Mock MCP Server，使用 Go `net/http` 提供单入口 `POST /mcp`。

当前状态：

- 可用于本地联调、协议回归和前端 viewport 联调
- `tools/*.yml` 是工具元数据与参数 schema 的唯一事实源
- 当前支持最小 `bash` 工具和 `tools/call.params._meta` 扩展

核心场景：

- 为 MCP 客户端、网关或智能体平台提供稳定的 mock 响应
- 在启动阶段校验工具定义与 Go handler 一致性
- 在调用阶段按 schema 校验 `tools/call.arguments`

## 技术栈

- 语言：Go `1.26`
- HTTP：标准库 `net/http`
- 配置与协议文件：`gopkg.in/yaml.v3`
- JSON Schema 校验：`github.com/santhosh-tekuri/jsonschema/v5`（通过本地 `third_party/jsonschema` 替换）
- 运行形态：本地 `go run` / Docker Compose / Docker 镜像
- 测试：Go 标准库 `testing`

## 架构设计

整体风格：单体 API 服务。

分层与职责：

- `cmd/mcp-server`
  - 组装配置、tool registry、viewport registry 与 HTTP controller
- `internal/config`
  - 维护代码默认值、内嵌 `application.yml`、外部 yml 与环境变量覆盖逻辑
- `internal/mcp/protocol`
  - 维护 JSON-RPC 请求/响应结构，包括 `tools/call._meta`
- `internal/mcp/spec`
  - 读取并校验 `tools/*.yml`
- `internal/mcp/schema`
  - 编译并执行 tool `inputSchema` 校验
- `internal/mcp/tools`
  - 维护 handler 接口、registry、最小 bash 执行器和 mock 工具实现
- `internal/mcp/transport`
  - 处理 JSON-RPC method 分发、SSE 输出和错误映射
- `internal/viewport`
  - 维护 viewport 文件注册与读取
- `internal/observability`
  - 负责日志、脱敏和摘要截断

关键设计决策：

- `arguments` 和 `_meta` 分离：只有 `arguments` 参与 schema 校验
- `_meta` 是工具可选消费的扩展上下文，协议层只负责透传
- `bash` 保持严格单命令模式，不支持管道、重定向、命令替换等高级 shell 语法
- `bash` 当前是最小 mock 验证能力，不追求完整 shell 托管

## 目录结构

- `cmd/`
  - 程序入口
- `docs/`
  - MCP 协议与 viewport 协议文档
- `internal/config/`
  - 配置结构、加载逻辑、测试与内嵌默认配置
- `internal/mcp/`
  - 协议、spec、schema、传输控制器与工具实现
- `internal/observability/`
  - 日志与脱敏
- `internal/viewport/`
  - viewport 注册与装载
- `tools/`
  - 工具定义文件
- `viewports/`
  - html / qlc viewport 文件
- `third_party/`
  - 本地替换的第三方依赖

## 数据结构

核心运行时结构：

- `config.Config`
  - `ServerPort`
  - `ToolsSpecLocationPattern`
  - `ViewportsDir`
  - `HTTPMaxBodyBytes`
  - `Observability`
  - `Bash`
- `config.BashConfig`
  - `WorkingDirectory`
  - `AllowedRoots`
  - `AllowedCommands`
  - `TimeoutMs`
  - `MaxCommandChars`
  - `MaxOutputChars`
- `protocol.ToolsCallParams`
  - `Name`
  - `Arguments`
  - `Meta`
- `tools.ToolCall`
  - `Arguments`
  - `Meta`
  - `WorkDirectory`
  - `UserID`

数据流：

1. 启动时加载配置
2. 读取 `tools/*.yml` 并校验定义
3. 编译每个工具的 `inputSchema`
4. 建立 spec 与 handler 注册表
5. `/mcp` 请求进入 controller 后按 method 分发
6. `tools/call` 先校验 `arguments`，再把 `arguments + _meta` 传给 handler
7. `bash` handler 当前会从 `_meta` 中消费工作目录和调用人身份

## API 定义

入口：

- `POST /mcp`

支持方法：

- `initialize`
- `tools/list`
- `tools/call`
- `viewports/list`
- `viewports/get`

`tools/call` 约定：

- 入参 `params` 包含 `name`、`arguments` 和可选 `_meta`
- `_meta` 是扩展上下文，不参与 `inputSchema` 校验
- 非消费 `_meta` 的工具可直接忽略它

`tools/list` 扩展字段：

- `label`
- `toolAction`
- `toolType`
- `viewportKey`

`viewports/*` 约定：

- `viewports/list` 返回 viewport 摘要列表
- `viewports/get` 返回 `viewportKey`、`viewportType`、`payload`
- 缺少或传入不存在的 `viewportKey` 时返回 `-32602`

错误码：

- `-32700`：parse error
- `-32600`：invalid request
- `-32601`：method not found
- `-32602`：invalid params
- `-32603`：internal error

协议详见：

- [docs/mcp-protocol-definition.md](/Users/linlay-macmini/Project/mcp-server-mock/docs/mcp-protocol-definition.md)
- [docs/viewport-protocol-definition.md](/Users/linlay-macmini/Project/mcp-server-mock/docs/viewport-protocol-definition.md)

## 开发要点

- 新增工具时必须同时更新 `tools/<tool>.yml` 与 `BuiltinHandlers(...)`
- 业务参数规则以 YAML `inputSchema` 为准，不在 Go 代码中重复维护
- `_meta` 不是 schema 校验对象；新增字段时优先保持向后兼容
- `bash` 只允许白名单命令，并限制在允许根目录内执行
- `bash` 对工作目录的消费方式是：当工具调用提供对应 `_meta` 上下文时，用它覆盖默认工作目录；归一化后仍必须落在允许根目录内
- `bash` 对调用人身份的消费方式是：透传到执行结果，并注入子进程环境变量 `MCP_USER_ID`；不会切换系统用户
- 配置职责边界：
  - `internal/config/application.yml`：内嵌默认配置
  - `configs/*.yml`：可选外部结构化配置
  - `.env`：本地键值与可选 `CONFIG_PATH`
  - 环境变量：最终覆盖层

## 开发流程

当前仓库可见流程：

- 本地运行：`make run`
- 测试：`make test`
- Docker 编排：`make docker-up`
- 镜像构建：`make docker-build`
- 国内网络构建：`make docker-build-cn` / `make docker-up-cn`

## 已知约束与注意事项

- `bash` 当前是“最小 mock 验证能力”，不是完整 shell 托管服务
- `bash` 不支持管道、重定向、here-doc、命令替换、变量替换或变量存储
- `workDirectory` 只影响 `bash` 的执行目录，不改变其他工具行为
- `workDirectory` 覆盖默认工作目录后仍必须通过允许根目录校验
- `userId` 只作为透传信息和子进程环境变量，不切换 Unix 用户
- `docs/` 维护的是本仓库协议契约，不是上游 MCP 规范原文镜像
