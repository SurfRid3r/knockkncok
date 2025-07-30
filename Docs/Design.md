# knockknock-NG —— 单包授权框架

---

## 0. 总览

| 组件               | 支持平台                              |
| ---------------- | --------------------------------- |
| 客户端 **`kk`**     | macOS / Linux / Windows / Android |
| 服务端 **`knockd`** | Linux / Windows                   |

场景：在服务器上运行无密码 SOCKS5 / RDP / SSH 等服务。外部主机须先发送一颗加密 TCP‑SYN（SPA）包“敲门”，`knockd` 验证通过后，把 **发包 IP 对配置端口集合** 加入临时白名单，TTL 由服务端算法决定。

---

## 1. 设计原则

1. **客户端极简：** 只负责敲门；不携带端口或 TTL。
2. **服务端集中控制：** 放通端口列表与 TTL 全部在配置中声明，由动态算法调整。
3. **平台差异最小：** 仅 `Sniffer` 与 `Firewall` 两个接口有分支实现，其余纯 Go。
4. **无监听端口：** 通过在数据链路层进行只读抓包（Sniffing）来识别 SPA 请求，自身不开放任何端口，从而隐藏攻击面。
5. **代码量精简：** 可运行守护进程约 350 行 Go。

---

## 2. 核心工作流

`knockd` 的核心安全模型基于“不监听、只观察”的原则，它不直接接收任何连接请求，而是作为防火墙的动态策略管理器。

1.  **默认状态（防火墙关闭）**: 服务器防火墙默认拒绝外部对 `allow_ports` 中端口（如 SSH 22）的访问。此时，虽然 `sshd` 等服务在运行，但对外界不可见。

2.  **被动观察 (`Sniffing`)**: `knockd` 在后台运行，通过 `pcap` 等库在指定的 `iface` 网络接口上进行底层抓包。它被动地检查每一个流经网卡的数据包，但不响应任何网络流量。从外部进行端口扫描，不会发现任何与 `knockd` 相关的开放端口。

3.  **信号识别 (`SPA Verification`)**: 当客户端 `kk` 发送一个加密的 SPA 包时，`knockd` 会捕获到这个包。它会使用共享密钥解密并验证包的有效性（时间戳、HMAC 等）。

4.  **动态授权 (`Firewall Management`)**: 一旦验证成功，`knockd` 的唯一动作是**调用系统防火墙工具**（如 `iptables`, `nftables`, `netsh`），为发送 SPA 包的源 IP 地址，临时性地打开通往 `allow_ports` 中指定端口的访问权限。

5.  **会话建立**: 防火墙规则更新后，客户端现在可以直接与先前被屏蔽的服务（如 SSH）建立标准连接。

6.  **自动撤权**: 当预设的 TTL 到期后，`knockd` 会再次调用防火墙工具，移除该临时规则，关闭访问权限，使服务器恢复到安全的默认状态。

这种机制将“服务发现”与“服务访问”完全分离，极大地提高了安全性。


---

## 2. 配置文件示例 (`knockd.toml`)

```toml
iface        = "eth0"            # Windows 示例 "Ethernet"
allow_ports  = [1080, 22, 443]   # 统一放行端口集合
base_ttl_min = 10                # 基础 TTL (分钟)
max_ttl_min  = 1440              # 最大 TTL (分钟)
db_file      = "whitelist.db"

key_E = "BASE64…"                # 256‑bit 加密密钥
key_H = "BASE64…"                # 256‑bit HMAC 密钥
```

---

## 3. SPA 协议 v2（精简）

| 字段        | 字节 | 描述                                 |
| --------- | -- | ---------------------------------- |
| Version   | 1  | 固定 `0x02`                          |
| Timestamp | 4  | Unix 秒，用于重放检测                      |
| AgentID   | 8  | 设备哈希                               |
| Nonce     | 8  | 随机数                                |
| MAC       | 16 | HMAC‑SHA256(key\_H, CipherText) 截断 |
| IV        | 16 | AES‑256‑CTR IV                     |

*明文 21 B，经 AES‑CTR 加密后与 IV、MAC 组成 53 B 负载，按序写入 IP ID、TCP Seq/Ack/Win/UrgPtr、Options。*

---

## 4. 动态 TTL 算法

```go
// 成功敲门时调用
ttl := min(maxTTL,
           baseTTL * (1 << bits.Len(uint(score+1))))
```

* `score` 为该 `(agent_id, IP)` 历史成功次数，按天×0.5 衰减。
* TTL 由服务端单方面决定，客户端无法影响。

---

## 5. 服务端流程

1. 读取 `knockd.toml`。
2. 初始化 `Sniffer(iface)` 与 `Firewall(runtime.GOOS)`。
3. 循环抓包→`proto.Verify()`：若通过则计算 TTL 并 `Firewall.Add(ip, allow_ports, ttl)`。
4. 定时器在 TTL 到期后自动 `Firewall.Del(...)`。
5. 事件以 JSON 行写入 `knockd.log`。

**关键文件**

```
main.go      # knockd
sniffer.go   # gopacket 封装
firewall.go  # Linux / Windows 分支
proto.go     # SPA 编解码
ttl.go       # 动态 TTL 引擎
db.go        # bbolt 持久化
```

---

## 6. 客户端 CLI

```bash
# 初始化（一次性）
kk init

# 发送敲门包（无需指定端口和 TTL）
kk send -s 1.2.3.4
```

Android 调用：`KnockKnock.send(ctx, "1.2.3.4")`。

---

## 7. 安全要点

* **重放防御：** `Timestamp ±30 s` + `Nonce` 唯一性。
* **无监听端口：** 减少攻击面。
* **最小特权：** Linux 用 `setcap` 赋予 net raw/admin；Windows 需安装 Npcap 并允许 `netsh`。
* **持久白名单：** 使用 bbolt 保存得分与过期时间，守护进程重启后自动恢复。

---

## 8. 开发里程碑（无 CI/CD，仅个人演示）

| 阶段     | 目标                        | 预计  |
| ------ | ------------------------- | --- |
| **M0** | 协议编解码 + `kk init/send`    | 1 周 |
| **M1** | 通用 `Sniffer` 实现           | 1 周 |
| **M2** | `Firewall` Linux + TTL 引擎 | 1 周 |
| **M3** | `Firewall` Windows        | 1 周 |
| **M4** | Android `.aar` 与演示 APK    | 1 周 |

> *纯个人 Demo，无自动化 CI/CD 流程；编译、打包、发布均手动执行。*

---

## 9. 最小守护示例 (main.go)

```go
func main() {
    cfg := LoadConfig("knockd.toml")
    fw  := newFirewall(runtime.GOOS)
    sn  := NewSniffer(cfg.Iface)

    for pkt := range sn.C() {
        info, ok := proto.Verify(pkt, cfg.KeyE, cfg.KeyH)
        if !ok { continue }

        ttl := ttlEngine.Next(info.AgentID, info.IP)
        if err := fw.Add(info.IP, cfg.AllowPorts, ttl); err != nil {
            log.Println("add rule fail", err)
        }
    }
}
```
