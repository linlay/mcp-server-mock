# MCP 协议定义

## 1. 文档范围

本文档描述 `mcp-server-mock` 当前实现的 MCP 协议契约。它是本仓库的实现文档，不是上游 MCP 官方规范原文。

协议入口：

- `POST /mcp`

协议版本：

- `jsonrpc: "2.0"`
- `initialize.result.protocolVersion: "2025-06"`

内容协商：

- 默认返回 `application/json`
- 当请求头 `Accept` 包含 `text/event-stream` 时，返回单条 SSE 事件

## 2. 通用请求格式

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "tools/call",
  "params": {}
}
```

字段约定：

- `jsonrpc`：必须是 `"2.0"`
- `id`：请求标识，透传回响应
- `method`：MCP method
- `params`：method 对应参数对象；为空对象或缺省时按 method 自身处理

## 3. 通用响应格式

成功响应：

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {}
}
```

错误响应：

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "error": {
    "code": -32602,
    "message": "invalid params: ..."
  }
}
```

说明：

- unknown tool 不返回 RPC 级 `method not found`
- unknown tool 仍返回 `result.isError=true`

## 4. 支持的方法

### 4.1 `initialize`

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06",
    "capabilities": {},
    "clientInfo": {
      "name": "runner",
      "version": "0.0.1"
    }
  }
}
```

响应：

- `protocolVersion`
- `serverInfo.name`
- `serverInfo.version`
- `capabilities.tools.listChanged`

### 4.2 `tools/list`

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "method": "tools/list",
  "params": {}
}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": "2",
  "result": {
    "tools": []
  }
}
```

每个 tool 至少包含：

- `type`
- `name`
- `description`
- `inputSchema`

本仓库扩展字段：

- `label`
  - 工具的人类可读名称，适合直接用于中文或业务化展示
- `toolAction`
  - 当值为 `true` 时，表示消费方可将该工具视为 action 工具
- `toolType`
  - 用于声明该工具关联的前端呈现类型；当前仅在显式声明 frontend tool 时返回
- `viewportKey`
  - 用于声明该工具关联的 viewport 标识；消费方可据此再调用 `viewports/get`
- `afterCallHint`
  - 工具调用后的提示文本，供客户端或智能体平台做补充渲染引导

约束：

- `toolAction=true` 不能与 `toolType` / `viewportKey` 同时声明
- `toolType` 与 `viewportKey` 必须一起声明

### 4.3 `tools/call`

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "method": "tools/call",
  "params": {
    "name": "mock.weather.query",
    "arguments": {
      "city": "shanghai",
      "date": "2026-02-14"
    },
    "_meta": {
      "traceId": "trace-001"
    }
  }
}
```

字段约定：

- `name`：工具名，必填
- `arguments`：工具业务参数对象；参与 `inputSchema` 校验
- `_meta`：可选扩展元数据对象；不参与 `inputSchema` 校验

本仓库扩展：

- 支持 `params._meta`
- 未识别的 `_meta` 字段不会触发协议层报错
- `_meta` 是否被消费由具体工具决定

`tools/call` 成功响应结构：

```json
{
  "jsonrpc": "2.0",
  "id": "3",
  "result": {
    "structuredContent": {},
    "content": [
      {
        "type": "text",
        "text": "{}"
      }
    ],
    "isError": false
  }
}
```

字段约定：

- `structuredContent`：结构化结果
- `content`：当前统一返回单个 `text` block
- `isError`：是否是工具级错误
- `error`：仅在工具级错误时存在

## 5. `_meta` 约定

`_meta` 当前仅在 `tools/call.params` 定义。

约束：

- `_meta` 不参与 tool `inputSchema` 校验
- 未识别字段不会导致协议层报错
- `_meta` 是否生效由具体工具自行决定

## 6. SSE 响应

当 `Accept: text/event-stream` 时：

- HTTP 状态码仍为 `200`
- `Content-Type: text/event-stream`
- 响应体格式：

```text
data: {"jsonrpc":"2.0","id":"1","result":{...}}

```

当前实现只返回一条 SSE 事件，不持续推流。

## 7. 错误码

- `-32700`：parse error，JSON 无法解析
- `-32600`：invalid request，请求结构非法，例如空 body
- `-32601`：method not found，不支持的 method
- `-32602`：invalid params，参数结构错误或 schema 校验失败
- `-32603`：internal error，服务内部错误或 panic
