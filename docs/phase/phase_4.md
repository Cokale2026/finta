## Phase 4: MCP 集成 (3-4 天)

### 目标

完整实现 MCP (Model Context Protocol) 支持，能够加载和使用 MCP 服务器。

### 实现步骤

#### 4.1 MCP 客户端基础

**文件**: `pkg/mcp/client.go`

参考 Go MCP SDK，实现基础的 JSON-RPC 2.0 客户端。

核心方法：

- Initialize
- ListTools
- CallTool
- ListResources
- ReadResource

#### 4.2 Stdio Transport

**文件**: `pkg/mcp/transport/stdio.go`

实现通过 stdio 与 MCP 服务器通信。

#### 4.3 MCP Tool Adapter

**文件**: `pkg/mcp/adapter.go`

将 MCP 工具适配为 Finta 工具接口。

#### 4.4 Plugin Manager

**文件**: `pkg/mcp/manager.go`

管理多个 MCP 服务器，统一工具注册。

#### 4.5 配置支持

**文件**: `configs/default.yaml`

```yaml
mcp:
  servers:
    - name: filesystem
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-filesystem"
        - "/home/user/projects"

    - name: github
      transport: stdio
      command: npx
      args:
        - "-y"
        - "@modelcontextprotocol/server-github"
      env:
        GITHUB_TOKEN: ${GITHUB_TOKEN}
```

### Phase 4 完成标准

- ✅ MCP JSON-RPC 2.0 客户端实现
- ✅ Stdio transport 工作
- ✅ MCP 工具可以适配为 Finta 工具
- ✅ 可以从配置加载多个 MCP 服务器
- ✅ MCP 工具与内置工具无缝集成

---
