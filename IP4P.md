# FRP + IP4P 分支说明（frpc / frps 内网穿透）

本项目基于官方 [fatedier/frp](https://github.com/fatedier/frp) **v0.72.0** 源码，
在**客户端（frpc）** 上新增了 **IP4P 地址解析** 支持，其余功能与官方 v0.72.0 完全一致。

> 这是一次**最小化、可溯源**的二次开发：只新增了 `client/ip4p.go`（含单元测试）、
> 改动客户端拨号路径的 2 处调用，协议、配置格式、鉴权、加密等核心逻辑均未改动。
> 本分支**没有**对二进制做反检测/免杀改造，成品即是标准 frp 客户端，便于审计与自行编译。

---

## 1. 什么是 IP4P

IP4P（IP for P2P）用于在 NAT 穿透 / P2P 场景下，把**动态变化的公网 IPv4 地址 + 端口**
通告给客户端。frps 位于 NAT 之后，公网映射地址/端口由 STUN / natmap / lucky 等工具探测后
写入 DNS，frpc 启动时自动查出真实的 `IP:端口` 再连接，因此 `serverAddr` 填一个域名即可，
**不需要静态公网 IP，也不需要固定端口**。

本分支按以下**顺序尝试三种解析方式**，命中即返回：

| 优先级 | 方式 | 记录内容 | 说明 |
| --- | --- | --- | --- |
| 1 | **TXT 记录** | `base64("IPv4:端口")`，例如 `MjAzLjAuMTEzLjc6MTgyNDY=` 解码为 `203.0.113.7:18246` | natmap / lucky / DDNS 类部署的常用方式，也兼容明文 `IPv4:端口` |
| 2 | **AAAA 记录** | IP4P 编码的 IPv6 地址（`2001:0000::/16` 前缀，端口在字节 10-11，IPv4 在字节 12-15） | hevz/frp 原始 IP4P 方案 |
| 3 | 普通模式 | 无匹配记录 | 原样使用 `serverAddr` + `serverPort`，行为与官方 frp 完全相同 |

**关键点：解析成功后，记录里的端口会覆盖配置中的 `serverPort`。**
所以使用 IP4P 时配置文件里**可以不写 `serverPort`**（写了也会被覆盖），这是预期行为。

例如域名 `your.ip4p.example.com` 的 TXT 记录为 `MjAzLjAuMTEzLjc6MTgyNDY=`：

```
TXT: MjAzLjAuMTEzLjc6MTgyNDY=
  └─ Base64 解码 ─▶ 203.0.113.7:18246
                      └─ frpc 实际连接 203.0.113.7 的 18246 端口
```

客户端日志里会打印解析结果，便于排查：

```
[I] ip4p: resolved server address from TXT record, your.ip4p.example.com -> 203.0.113.7:18246
```

解析逻辑位于新增文件 [`client/ip4p.go`](client/ip4p.go) 的 `lookupIP4P()`，
并在 [`client/connector.go`](client/connector.go) 的 QUIC 与 TCP/WS 两条拨号路径中被调用。

---

## 2. 与原版 frp 的差异（仅客户端）

| 文件 | 改动 |
| --- | --- |
| `client/ip4p.go` | **新增**：`lookupIP4P(ctx, addr, port)`，支持 TXT(Base64/明文) 与 AAAA(IP4P IPv6) 两级解析，失败回退普通模式 |
| `client/connector.go` | QUIC 拨号处、TCP/WS 拨号处各插入一次 `lookupIP4P` 解析 `server_addr` |
| `client/ip4p_test.go` | **新增**：TXT 解析与回退行为的单元测试 |

服务端（frps）**不需要任何改动**，使用官方 v0.72.0 的 frps 即可。

---

## 3. 编译（从源码）

### 3.1 环境要求
- Go **1.24+**（本分支在 Go 1.26 下验证通过）
- 支持 Windows / Linux / macOS 交叉编译

### 3.2 编译 Windows x64 客户端（frpc）

```powershell
# 在项目根目录执行
$env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc.exe ./cmd/frpc/
```

- 标签 `frpc` 仅编译客户端相关代码；`noweb` 用于跳过内置 Web 资源（管理员页面）的嵌入，
  避免缺少前端构建产物时编译报错。如需带 Web 资源，请先按官方流程 `make web` 生成 `web/frpc/dist`。
- 编译产物的版本号执行 `frpc.exe --version` 查看，应为 `0.72.0`。

### 3.3 交叉编译其他平台

```bash
# Linux amd64
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 \
  go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc_linux_amd64 ./cmd/frpc/

# Linux arm64
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 \
  go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc_linux_arm64 ./cmd/frpc/

# macOS arm64
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 \
  go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc_darwin_arm64 ./cmd/frpc/

# 服务端 frps（任意平台）
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -tags "frps,noweb" -trimpath -ldflags="-s -w" -o bin/frps_linux_amd64 ./cmd/frps/
```

> 说明：Git Bash 下请勿用 `env GOOS=... go build` 前缀形式（在 Windows 上可能静默不产出文件），
> 直接用 `GOOS=... GOARCH=... go build` 前置赋值即可。

---

## 4. 配置示例

### 4.1 使用 IP4P（TXT 记录）的客户端 `frpc.toml`

```toml
serverAddr = "your.ip4p.example.com"   # 该域名的 TXT 记录存 base64("公网IP:端口")
# serverPort 可不写：解析出的端口会自动覆盖默认值

auth.method = "token"
auth.token  = "请把这里换成你自己的强令牌"

# 如需指定 DNS 服务器（例如本机 AdGuard Home / 内网 DNS）
# dnsServer = "192.168.1.1:53"

[[proxies]]
name      = "jl2"
type      = "stcp"
secretKey = "请把这里换成你自己的密钥"
localIP   = "127.0.0.1"
localPort = 3389
```

启动：`frpc.exe -c frpc.toml`

> `dnsServer` 会接管 Go 的 `net.DefaultResolver`，因此 TXT 查询也会走它。
> 若 TXT 查询失败，请确认该 DNS 服务器**没有过滤 TXT 记录**，并可用
> `nslookup -type=TXT your.ip4p.example.com` 自查。
> 用 `base64 -d` 或在线工具解码 TXT 值，应得到形如 `IP:端口` 的字符串。

### 4.2 普通模式（无 IP4P 记录，等价官方行为）

```toml
serverAddr = "your.frps.example.com"
serverPort = 7000
auth.token = "请把这里换成你自己的强令牌"

[[proxies]]
name = "ssh"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000
```

### 4.3 服务端 `frps.toml`（用官方 v0.72.0 的 frps）

```toml
bindPort = 7000
auth.token = "请把这里换成你自己的强令牌"
```

启动：`frps -c frps.toml`

更完整的配置项请参考官方文档与本项目根目录的 `README.md` / `README_zh.md`。

---

## 5. 关于杀毒软件"误报"的合法处理

frp 是合法的开源内网穿透工具，但部分杀毒软件会基于**特征签名**把它当作潜在风险程序（HackTool / RiskWare）拦截或删除。
这属于**误报**，正确、合规的处理方式是：

1. **代码签名**：用你自己的代码签名证书（EV 代码签名证书最佳）对二进制进行签名，
   可显著降低被拦截的概率，也便于在企业环境部署。
2. **加入白名单 / 排除项**：在 Windows Defender、火绒、360 等软件中，
   把你自编译的 `frpc.exe` 所在目录加入"排除项 / 信任区"。
3. **向厂商提交误报**：把你的二进制哈希提交给 Microsoft 等厂商申诉：
   https://www.microsoft.com/en-us/wdsi/filesubmission
4. **仅在你拥有或获授权的设备上使用**，不要把客户端部署到任何未经许可的机器上。

> 说明：本分支**刻意不做**任何"反检测 / 免杀"改造。刻意改造二进制以躲避终端安全软件、
> 并对外分发的行为，容易被用于在未授权设备上隐蔽部署，存在法律与安全风险，因此本仓库不提供此类改动。

---

## 6. 许可证与归属

- 整体基于 [fatedier/frp](https://github.com/fatedier/frp) v0.72.0，遵循 **Apache License 2.0**。
- IP4P 地址解码实现移植自 [hevz/frp](https://github.com/hevz/frp)（ip4p 分支），该文件保留原作者版权声明（Apache 2.0）。
- TXT 记录解析方式参考 [qaz741wsd856/frp](https://github.com/qaz741wsd856/frp)。
- 使用、修改、再分发请遵守 Apache 2.0 许可证，并保留相应版权与许可声明。

---

## 7. 免责声明

本软件仅供在**你拥有或获得明确授权**的设备与网络上进行内网穿透 / 远程访问使用。
使用者须自行遵守所在地法律法规，因滥用本软件造成的任何后果由使用者自行承担。
