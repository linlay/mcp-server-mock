# Viewport 协议定义

## 1. 文档范围

viewport 是 `mcp-server-mock` 提供的配套协议能力，接口形式接近 MCP，但独立使用 `viewports/*` method 暴露。

目标：

- 让工具结果能够关联 html 或 qlc 视图定义
- 为前端或智能体平台提供统一的视图拉取接口

## 2. 文件模型

viewport 文件目录默认是 `./viewports`。

支持文件类型：

- `.html`
  - 以文件名去掉扩展名作为 `viewportKey`
  - `payload` 返回字符串
- `.qlc`
  - 以文件名去掉扩展名作为 `viewportKey`
  - 文件内容必须是合法 JSON
  - `payload` 返回对象

示例：

- `show_weather_card.html`
- `todo_form.qlc`

## 3. `viewports/list`

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "8",
  "method": "viewports/list",
  "params": {}
}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": "8",
  "result": {
    "viewports": [
      {
        "viewportKey": "show_weather_card",
        "viewportType": "html"
      }
    ]
  }
}
```

字段说明：

- `viewportKey`
- `viewportType`

## 4. `viewports/get`

请求：

```json
{
  "jsonrpc": "2.0",
  "id": "9",
  "method": "viewports/get",
  "params": {
    "viewportKey": "show_weather_card"
  }
}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": "9",
  "result": {
    "viewportKey": "show_weather_card",
    "viewportType": "html",
    "payload": "<html>...</html>"
  }
}
```

字段说明：

- `viewportKey`
- `viewportType`
- `payload`

`payload` 类型：

- `html`：字符串
- `qlc`：对象

## 5. 错误约定

`viewports/get` 在以下场景返回 `-32602`：

- 缺少 `viewportKey`
- 指定的 `viewportKey` 不存在

registry 启动阶段还会校验：

- 若工具声明了某个 `viewportKey`，但 `viewports/` 中不存在对应文件，则服务启动失败
- 若同名 viewport 文件重复映射同一 key，则服务启动失败

## 6. 与 MCP 的关系

相同点：

- 都走同一个 `/mcp` JSON-RPC 入口
- 都使用 `jsonrpc/id/method/params` 结构
- 都遵循统一错误码体系

不同点：

- `tools/*` 面向工具执行
- `viewports/*` 面向视图定义发现与拉取
- viewport 不参与 `inputSchema` 校验
