# Settings + .env Persistence Refactor

## 改动范围

| 文件 | 操作 |
|---|---|
| `tui/tui.go` | 新增 `ViewSettings`、`settingsPanel` 字段、`OnSaveSettings` 回调 |
| `tui/settings.go` | **新建**：配置管理视图 |
| `tui/menu.go` | 菜单第一项改为"配置管理"，索引调整 |
| `main.go` | 移除 `fmt.Scanln` 阻塞提示，新增 `.env` 自动创建/保存 |

## 设计

### 启动流程

```
main()
  → godotenv.Load()
  → ensureEnvFile()    // .env 不存在则创建空模板
  → os.Getenv() 读取   // 静默加载，无阻塞
  → tea.NewProgram()   // 直接进入 TUI
```

### 配置管理视图

```
┌─ 配置管理 ──────────────────────────────┐
│                                         │
│  API Key:  [sk-xxxxxxxxxxxxxxxx****]    │
│  AppId:    [1f03bff2a0f74eae9e1b553f]  │
│                                         │
│  enter 保存并应用  esc 返回             │
└─────────────────────────────────────────┘
```

- 两个 `bubbles/textinput` 单行输入
- API Key 输入框设为密码模式（`EchoMode: EchoPassword`）或显示为掩码
- 保存时写入 `.env` 文件，同时更新 `Model.apiKey/appId`
- 如果 apiKey/appId 为空，其他功能（对话、批处理）运行时提示"请先配置 API Key 和 AppId"

### 主菜单变化

```
1. 配置管理          ← 新增，取代原来的启动提示
2. 开始/继续对话
3. 新对话
4. 规则模式
5. 退出
```

### .env 持久化

```go
func saveEnvFile(apiKey, appId string) error {
    // 读取已有内容，更新 LLM_API_KEY / LLM_APP_ID 行，写回
}
```

直接用 `godotenv.Write()` 或逐行替换。保持已有的 ALIBABA_CLOUD_ACCESS_KEY_ID 等字段不变。

### 防护

- 对话/批处理启动时检查 `apiKey == ""`，若为空提示"请先在配置管理中设置 API Key"
- 批处理启动时检查 `appId == ""`，若为空提示"请先在配置管理中设置 AppId"

## 验证

1. `go build ./...` 编译通过
2. 首次运行：自动创建 `.env`，进入配置管理可输入保存
3. 保存后对话/批处理正常使用
4. 再次启动：自动读取已有 `.env`，无需重新输入
