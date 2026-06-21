# filebrowser-cli

> **Go ≥ 1.21 要求** | 跨平台：Windows / Linux / macOS

**⚠️ 本项目无 UI**

filebrowser-cli 是被 skill / agent 调用的 shell 工具，可观察入口在 stdout / stderr / 退出码。无 Web UI / TUI / 图形界面。

## 概述

filebrowser-cli 提供了对 FileBrowser HTTP API 的完整命令行封装，支持文件管理、分享、搜索等全部功能。

## 项目结构

```
filebrowser-cli/
├── cmd/                  # 命令定义
│   ├── root.go           # 根命令
│   ├── login.go          # 登录命令
│   ├── ls.go             # 列出目录
│   ├── upload.go         # 上传文件
│   ├── download.go       # 下载文件
│   ├── share.go          # 分享命令
│   ├── search.go         # 搜索命令
│   └── ...               # 其他命令
├── internal/             # 私有实现
│   ├── client/           # API 客户端
│   └── config/           # 配置管理
├── pkg/                  # 可复用的公共包
│   ├── config/           # 配置加载
│   ├── httpclient/       # HTTP 客户端
│   ├── output/           # 输出格式化
│   └── version/          # 版本信息
├── integration/          # 集成测试
├── main.go               # 入口文件
├── go.mod
├── Makefile
└── README.md
```

## Windows Git Bash 兼容性

在 Git Bash / MSYS2 环境下运行时，shell 会在将参数传给程序之前自动把 POSIX 路径（如 `/`、`/foo`）转换为 Windows 路径（如 `C:/Program Files/Git/`）。这会导致远程服务器路径被错误解析。

filebrowser-cli 会在 Windows + MSYS 环境下自动检测并反转这种路径转换，使 `filebrowser-cli ls /` 在 Git Bash 中也能正确列出服务器根目录。

如需彻底避免转换，可在 shell 中设置环境变量：

```bash
# Git Bash / MSYS2 — 禁止所有路径转换
export MSYS_NO_PATHCONV=1

# 或者仅排除 filebrowser-cli 的参数
export MSYS2_ARG_CONV_EXCL="*"
```

## 安装

### 从源码安装

```bash
git clone https://github.com/ANIAN0/filebrowser-cli.git
cd filebrowser-cli
go build -o filebrowser-cli .
```

### 使用 go install

```bash
go install github.com/ANIAN0/filebrowser-cli@latest
```

### 使用 Makefile

```bash
make install
```

### Release 二进制自管理

从 GitHub Release 下载的二进制不需要 Go 环境即可安装、更新和卸载：

```bash
# 安装当前二进制到默认用户 bin 目录
filebrowser-cli install

# 安装到指定目录
filebrowser-cli install --dir /path/to/bin

# 从最新 GitHub Release 更新当前二进制
filebrowser-cli update

# 即使当前版本相同也强制重新下载
filebrowser-cli update --force

# 卸载二进制，默认保留用户配置
filebrowser-cli uninstall

# 卸载二进制并删除用户配置目录
filebrowser-cli uninstall --purge
```

默认安装目录：

- Linux/macOS: `~/.local/bin`
- Windows: `%LOCALAPPDATA%\Programs\filebrowser-cli`

如果安装目录不在 `PATH` 中，`install` 会输出提示。

## 配置

### 配置文件位置（按优先级）

1. `--config <path>` 命令行参数
2. `FILEBROWSER_CLI_CONFIG` 环境变量
3. 二进制同级目录的 `config.yaml`（项目安装模式）
4. 用户目录：`~/.config/filebrowser-cli/config.yaml`（Unix）或 `%APPDATA%\filebrowser-cli\config.yaml`（Windows）

### 配置文件示例

```yaml
version: 1
instance_url: "http://localhost:8080"
username: "admin"
password: "${FB_PASSWORD}"  # 支持环境变量插值
default_expires: 24
default_unit: "hours"
```

### 环境变量插值

配置文件中支持 `${ENV_VAR}` 格式的环境变量插值：

```yaml
password: "${FB_PASSWORD}"
```

运行前设置环境变量：

```bash
export FB_PASSWORD="your-password"
```

## 子命令

### 认证（Auth）

| 命令 | 说明 |
|------|------|
| `login` | 登录并获取 token |
| `renew` | 续期 token |
| `whoami` | 显示当前用户信息 |

**Token 自动续期**：JWT token 具有过期时间（通常 2 小时）。当 token 过期后，如果配置文件中设置了 `username` 和 `password`，工具会自动使用凭据重新登录，无需手动干预。若未配置凭据，则会提示运行 `filebrowser-cli login`。

```
# config.yaml 中配置凭据即可启用自动重登录
instance_url: "http://localhost:8080"
username: "admin"
password: "${FB_PASSWORD}"  # 支持环境变量
```

### 资源管理（Resource）

| 命令 | 说明 |
|------|------|
| `ls <path>` | 列出目录内容 |
| `tree <path>` | 显示目录树 |
| `info <path>` | 显示文件/目录详细信息 |
| `upload <local> [remote]` | 上传文件 |
| `download <remote> [local]` | 下载文件 |
| `mkdir <path>` | 创建目录 |
| `rm <path>` | 删除文件/目录 |
| `mv <src> <dst>` | 移动/重命名 |
| `cp <src> <dst>` | 复制 |

### 预览（Preview）

| 命令 | 说明 |
|------|------|
| `preview <path> [--size thumb\|big]` | 预览图片 |

### 分享（Share）

| 命令 | 说明 |
|------|------|
| `share create <path>` | 创建分享链接 |
| `share list` | 列出所有分享 |
| `share delete <hash>` | 删除分享 |
| `share info <path>` | 查看分享信息 |

### 搜索（Search）

| 命令 | 说明 |
|------|------|
| `search <path> <query>` | 搜索文件 |

## 全局选项

| 选项 | 说明 |
|------|------|
| `--config <path>` | 指定配置文件路径 |
| `--json` | 输出 JSON 格式（可被 `jq` 解析） |
| `--verbose, -v` | 详细日志到 stderr |
| `--timeout <seconds>` | HTTP 请求超时（默认 60） |
| `--no-color` | 禁用颜色输出 |
| `--version` | 输出版本信息 |
| `--help, -h` | 显示帮助 |

### 生命周期命令

| 命令 | 说明 |
|------|------|
| `install [--dir <path>]` | 将当前二进制安装到用户 bin 目录 |
| `update [--force]` | 从最新 GitHub Release 下载并替换当前二进制 |
| `uninstall [--dir <path>] [--purge]` | 删除已安装二进制；`--purge` 同时删除用户配置 |

### 配置与补全

| 命令 | 说明 |
|------|------|
| `config path` | 显示配置搜索路径 |
| `config init [--path <path>] [--force]` | 创建示例配置文件 |
| `config validate` | 验证当前配置 |
| `config show [--redact=false]` | 显示当前配置，默认隐藏敏感字段 |
| `completion bash\|zsh\|fish\|powershell` | 生成 shell completion 脚本 |

## 退出码约定

| 退出码 | 含义 | 触发条件 |
|--------|------|----------|
| `0` | 成功 | 请求成功（2xx） |
| `1` | 客户端错误 | HTTP 4xx（401/403/404/409） |
| `2` | 服务端错误 | HTTP 5xx（500/502/503/504） |
| `3` | 网络错误 | DNS 失败、连接超时、连接拒绝 |
| `4` | 配置错误 | 配置文件不存在、字段缺失、环境变量未设置 |

错误详情输出到 stderr，成功数据输出到 stdout。

## 使用示例

```bash
# 登录
filebrowser-cli login

# 列出根目录
filebrowser-cli ls /

# 上传文件
filebrowser-cli upload ./report.pdf /documents/report.pdf

# 创建分享链接
filebrowser-cli share create /documents/report.pdf --expires 24 --unit hours

# 搜索文件
filebrowser-cli search / "report"

# 使用 JSON 输出
filebrowser-cli ls / --json | jq '.items'
```

## 开发

```bash
# 运行测试
make test

# 清理构建产物
make clean
```

## 路径参数语义

位置参数为**远端 POSIX 路径**，约定：

- 必须以 `/` 开头（表示远端 server 上的绝对路径）。
- 包含空格、`#`、`?`、`%` 等字符时会自动按 URL 段编码。
- 在 Windows + Git for Windows（Git Bash）环境下，如果 argv 中的 `/` 被 shell 在传递前改写为 MSYS 安装根（例如 `C:/Program Files/Git/`），CLI 内部会自动反向还原。这一处理在交互式 shell 与非交互式 `bash -c` 环境都生效。
- `download <remote> [local]`：本地参数缺省时取远端路径的 basename（POSIX `path.Base` 语义，跨平台一致）。

## config 字段契约

`config.yaml` 与 `config show --json` 使用统一的 snake_case 字段名（YAML key / JSON key / Go 字段名一一对应）：

| YAML | JSON |
|---|---|
| `instance_url` | `instance_url` |
| `username` | `username` |
| `password` | `password` |
| `default_expires` | `default_expires` |
| `default_unit` | `default_unit` |
| `version` | `version` |

`config show` 默认 `--redact=true`，`password` 输出为 `***`；显式 `--redact=false` 输出真实值（仅诊断场景使用）。

## 许可证

MIT License
