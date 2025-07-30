# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 提供在此代码库中工作的中文指导。

## 项目概览

`knockknock-NG` 是一个单包授权（SPA）框架，包含极简客户端（`kk`）和服务端守护进程（`knockd`）。通过加密的 TCP SYN 数据包进行身份验证，无需开放端口即可安全访问 SSH、SOCKS5、RDP 等服务。

## 架构设计

- **客户端 (`kk/`)**：用于发送 SPA 数据包的命令行工具
- **服务端 (`knockd/`)**：被动嗅探流量并管理防火墙规则的守护进程
- **协议**：采用 AES-256-CTR 加密的 SPA 数据包，配合 HMAC-SHA256 身份验证
- **存储**：使用 bbolt 嵌入式数据库进行持久化白名单管理

## 构建命令

```bash
# 构建当前平台版本
go build -o output/knockd ./knockd
go build -o output/kk ./kk

# 交叉编译示例
GOOS=linux GOARCH=amd64 go build -o output/knockd ./knockd
GOOS=windows GOARCH=amd64 go build -o output/knockd.exe ./knockd
GOOS=darwin GOARCH=amd64 go build -o output/kk ./kk
```

## 使用说明

### 服务端设置

1. 创建 `knockd.toml` 配置文件
2. 运行：`./knockd`（需要 root/管理员权限访问防火墙）

### 客户端设置

1. 初始化：`./kk init`（生成主密钥）
2. 发送敲门包：`./kk send -s <服务器IP> -k <主密钥>`

## 核心组件

- `knockd/main.go:77` - 主数据包处理循环
- `knockd/firewall.go` - 平台特定的防火墙实现
- `knockd/sniffer.go` - 使用 gopacket 进行网络数据包捕获
- `knockd/proto.go` - SPA 数据包编码/解码
- `knockd/ttl.go` - 基于客户端历史的动态 TTL 计算
- `kk/main.go:10` - 客户端操作 CLI 接口

## 依赖项

- `github.com/google/gopacket` - 网络数据包处理
- `go.etcd.io/bbolt` - 嵌入式数据库
- `github.com/BurntSushi/toml` - 配置文件解析

## 安全模型

- 服务端无监听端口（仅被动嗅探）
- 通过时间戳和随机数防止重放攻击
- 动态防火墙规则，自动过期
- 基于评分机制的持久化白名单 TTL 计算
