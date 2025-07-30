# knockknock-NG

`knockknock-NG` 是一个为极简和安全而设计的单包授权（SPA）框架。

## 0. 总览

- **客户端 (`kk`)**: 一个适用于 macOS、Linux、Windows 和 Android 的极简命令行工具。
- **服务端 (`knockd`)**: 一个适用于 Linux 和 Windows 的守护进程。

主要使用场景是在服务器上运行无密码的 SOCKS5、RDP 或 SSH 等服务。外部主机必须先发送一个经过特殊加密的 TCP-SYN 包（即“敲门”），服务端验证通过后，会将发送该包的 IP 地址加入临时白名单，从而授权访问预设的端口集合。

## 1. 设计原则

- **客户端极简**: 只负责发送敲门包，不关心端口或 TTL。
- **服务端集中控制**: 所有放行端口和 TTL 均在服务端配置，由动态算法调整。
- **平台差异最小**: 代码为纯 Go 实现，仅 `Sniffer` 和 `Firewall` 两个接口针对不同操作系统有专门实现。
- **无监听端口**: 通过底层抓包（raw socket）识别 SPA 请求，不开放任何端口，减少攻击面。
- **代码量精简**: 核心守护进程的代码量约 350 行。

## 工作原理：隐形的守门人

`knockd` **不会开放任何端口来监听连接**，这正是其安全性的核心。它的工作模式更像一个通过隐藏摄像头观察交通的隐形守门人。

1. **敲门之前**: 服务器的防火墙默认阻止外界访问敏感端口（例如 SSH 的 22 端口）。SSH 服务本身在运行，但外界无法访问。

2. **敲门**: 您在客户端执行 `kk send`，这会向服务器发送一个经过加密的、特殊的数据包。这个包就像一个秘密的接头暗号。

3. **守门人观察**: 在后台运行的 `knockd` 正在被动地监听网络接口上的所有流量。它会捕获到这个特殊的“暗号包”，并用您预设的密钥来解密和验证它。

4. **开门**: 如果暗号正确，`knockd` 的唯一工作就是命令服务器的防火墙：“请为刚刚发送暗号的那个 IP 地址，临时允许访问 22 端口。”

5. **连接**: 防火墙上有了一个专为您开放的临时通道。您现在可以立刻使用标准的 SSH 客户端 (`ssh user@your_server`) 成功连接。其他任何人尝试连接，依然会被防火墙阻挡。

6. **关门**: 当配置的 TTL (有效时间) 到期后，`knockd` 会自动命令防火墙移除该临时规则，大门再次关闭，直到下一次成功的敲门。

这个机制意味着您的服务器几乎没有暴露任何攻击面，因为受保护的服务在您成功敲门之前是完全“隐形”的。

## 2. 快速开始

### 服务端 (`knockd`)

1. **创建配置文件 (`knockd.toml`)**:

    ```toml
    iface        = "eth0"            # (可选) 指定监听的网络接口。如果留空，程序会自动检测。
    allow_ports  = [1080, 22, 443]   # 成功敲门后放行的端口
    base_ttl_min = 10                # 基础 TTL (分钟)
    max_ttl_min  = 1440              # 最大 TTL (分钟)
    db_file      = "whitelist.db"

    key = "BASE64…"                # 256-bit 主密钥
    ```

    如果未指定 `key`，服务端将自动生成一个新的密钥并将其打印到控制台。

2. **运行服务端**:

    ```bash
    ./knockd
    ```

### 客户端 (`kk`)

1. **初始化客户端 (一次性操作)**:

    ```bash
    ./kk init
    ```

    该命令会生成主密钥（`key`），请将它复制到服务端的 `knockd.toml` 配置文件中。

2. **发送敲门包**:

    ```bash
    ./kk send -s <服务器IP> -k <主密钥>
    ```

## 从源码编译

您需要先安装 Go 环境才能从源码编译 `knockd` 和 `kk`。您可以使用交叉编译功能为不同的操作系统生成可执行文件。

所有编译后的二进制文件都将存放在 `output` 目录下。

### 编译到 Linux (amd64)

```bash
# 服务端
GOOS=linux GOARCH=amd64 go build -o output/knockd ./knockd

# 客户端
GOOS=linux GOARCH=amd64 go build -o output/kk ./kk
```

### 编译到 Windows (amd64)

```bash
# 服务端
GOOS=windows GOARCH=amd64 go build -o output/knockd.exe ./knockd

# 客户端
GOOS=windows GOARCH=amd64 go build -o output/kk.exe ./kk
```

### 编译到 macOS (amd64)

```bash
# 客户端
GOOS=darwin GOARCH=amd64 go build -o output/kk ./kk
```

## 7. 安全要点

### 程序权限

`knockd` 需要足够的权限来修改防火墙规则。

- **在 Linux 上**: 您必须以 `root` 用户身份运行 `knockd`，或者为其授予所需的网络权限：

    ```bash
    sudo setcap cap_net_raw,cap_net_admin+eip ./output/knockd
    ```

- **在 Windows 上**: 您必须以 **管理员权限** 运行 `knockd.exe`。

- **重放防御**: SPA 包包含时间戳和 Nonce，可有效防止重放攻击。
- **无监听端口**: 服务端不监听任何端口，极大减少了攻击面。
- **最小特权**: 服务端以最小的权限运行。
- **持久化白名单**: 使用 `bbolt` 数据库存储白名单，重启后可自动恢复。
