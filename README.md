# LLM Util

基于阿里云百炼平台的大模型批量查询工具，提供 TUI 终端界面，支持 Excel 驱动的批量调用。

> 仅支持 Agent 1.0（旧版智能体应用）

## 功能特性

- **TUI 终端界面** — 基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 构建，交互友好
- **模式 A：纯文本批量请求** — 读取 Excel 中的问题列表，批量调用百炼应用接口
- **模式 B：带文件批量请求** — 读取文件 + 问题，批量调用百炼应用接口（支持 PDF、Word 等）
- **断点续传** — 状态为"完成"的行自动跳过，失败行自动重试，中断后无需从头开始
- **并发控制** — 可配置并发数（默认 4，最大 200），基于 [ants](https://github.com/panjf2000/ants) 协程池
- **进度可视化** — 实时进度条 + 旋转指示器 + 成功/失败/跳过统计
- **结构化日志** — 基于 `log/slog` 输出到文件，方便排查问题
- **配置持久化** — 运行时修改的配置自动保存到 `.env` 文件

## 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.26 |
| TUI 框架 | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Bubbles](https://github.com/charmbracelet/bubbles) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| AI 平台 | 阿里云百炼（旧版智能体 API） |
| Excel 处理 | [excelize](https://github.com/xuri/excelize) |
| 并发池 | [ants](https://github.com/panjf2000/ants) |
| 配置管理 | [godotenv](https://github.com/joho/godotenv) |

## 快速开始

### 前置条件

- Go 1.26+
- 阿里云百炼账号，已创建 Agent 1.0 应用并获取 AppID 和 API Key

### 安装与运行

```bash
# 克隆项目
git clone <repo-url>
cd llm-util

# 复制配置文件并填入凭据
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY 和 LLM_APP_ID

# 运行
go run main.go
```

也可以在 TUI 的「配置管理」中交互式输入 API Key 和 AppID。

### 构建

**Windows：**
```cmd
build.bat
```

**Linux/macOS（交叉编译 Windows）：**
```bash
chmod +x build.sh
./build.sh
```

构建产物为 `llm-util.exe`（Windows amd64）。

如需自定义输出名称：
```bash
set BIN_NAME=my-tool && build.bat
# 或
BIN_NAME=my-tool ./build.sh
```

## 配置说明

配置文件 `.env` 支持以下选项：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `LLM_API_KEY` | 百炼 API Key（必填） | — |
| `LLM_APP_ID` | 百炼应用 AppID（必填） | — |
| `WORKSPACE_ID` | 子业务空间 ID | `llm-shwq55idtv5plnag` |
| `POOL_SIZE` | 批量处理并发数 | `4` |
| `ALIBABA_CLOUD_ACCESS_KEY_ID` | 阿里云 AK（文件上传用） | — |
| `ALIBABA_CLOUD_ACCESS_KEY_SECRET` | 阿里云 SK（文件上传用） | — |

所有配置项也可在 TUI「配置管理」面板中修改，保存后自动写入 `.env`。

## 使用方式

### 模式 A — 纯文本批量请求

1. 在 TUI 中选择 **批处理 → 模式A**
2. 点击 **模板下载**，生成 `template-A-*.xlsx`
3. 在 Excel 的 **request** 列（A 列）填入问题
4. 选择 **运行任务**，选择 Excel 文件执行

**Excel 模板格式：**

| A (request) | B (response) | C (status) | D (time) | E (errMsg) |
|---|---|---|---|---|
| 提问内容 | AI 填写 | 自动 | 自动 | 自动 |

### 模式 B — 带文件批量请求

1. 在 TUI 中选择 **批处理 → 模式B**
2. 点击 **识别文件目录**，选择包含目标文件的文件夹
3. 点击 **模板下载**，生成 `template-B-*.xlsx`（fileName 列自动填充）
4. 在 **request** 列（A 列）填入问题
5. 选择 **运行任务**，选择 Excel 文件执行

**Excel 模板格式：**

| A (request) | B (fileName) | C (response) | D (status) | E (time) | F (errMsg) |
|---|---|---|---|---|---|
| 提问内容 | 自动填充 | AI 填写 | 自动 | 自动 | 自动 |

### 断点续传

两种模式均支持断点续传：
- 状态列为 **"完成"** 的行自动跳过
- 失败行重新处理
- 每处理 10 条自动保存一次进度

## 项目结构

```
llm-util/
├── ai/                     # 百炼 API 客户端封装
├── conf/                   # 配置结构体与默认值
├── file/                   # 文件上传（租约机制）
├── internal/app/           # 业务逻辑（批处理引擎）
├── tui/                    # TUI 界面（Bubble Tea）
├── util/                   # 工具函数
├── main.go                 # 入口
├── build.bat               # Windows 构建脚本
└── build.sh                # Linux/macOS 构建脚本
```

## 日志

运行日志输出到当前目录的日志文件，基于 Go 标准库 `log/slog` 结构化日志。

日志级别：INFO（正常运行）、ERROR（任务失败/异常）。
