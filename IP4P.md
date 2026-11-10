# FRP + IP4P 分支说明（frpc / frps 内网穿透）

本项目基于官方 [fatedier/frp](https://github.com/fatedier/frp) **v0.72.0** 源码，
在**客户端（frpc）** 上新增了 **IP4P 地址解析** 支持，其余功能与官方 v0.72.0 完全一致。

> 这是一次**最小化、可溯源**的二次开发：只新增了 1 个文件、改动客户端拨号路径的 2 处调用，
> 协议、配置格式、鉴权、加密等核心逻辑均未改动。本分支**没有**对二进制做反检测/免杀改造，
> 成品即是标准 frp 客户端，便于审计与自行编译。

---

## 1. 什么是 IP4P

IP4P（IP for P2P）是一种把 **IPv4 地址 + 端口** 编码进 **IPv6 地址** 的表示方式，
前缀为 `2001:0000::/16`。常用于 P2P / NAT 穿透场景：当一个对端（peer）没有公网 IPv4，
但可以通过某种方式通告一个 IPv6 形式的 "IPv4:端口" 时，客户端在连接前把它解码回真实的
IPv4 与端口即可直连。

在本分支中，`server_addr` 既可以填普通域名 / IPv4，也可以填一个 **IP4P 编码的 IPv6 地址**。
客户端在发起连接（TCP / WebSocket / QUIC 三种传输方式）前会自动做一次解码：

```
2001:0000::IPv4:PORT  ──解码──▶  IPv4 地址 + 端口
```

如果 `server_addr` 不是 IP4P 地址，则原样连接，行为与普通 frp 完全相同。

解码逻辑位于新增文件 [`client/ip4p.go`](client/ip4p.go) 的 `lookupIP4P()`，
并在 [`client/connector.go`](client/connector.go) 的 QUIC 与 TCP/WS 两条拨号路径中被调用。

---

## 2. 与原版 frp 的差异（仅客户端）

| 文件 | 改动 |
| --- | --- |
| `client/ip4p.go` | **新增**：`lookupIP4P(addr, port)` 解码 IP4P 地址 |
| `client/connector.go` | QUIC 拨号处、TCP/WS 拨号处各插入一次 `lookupIP4P` 解析 `server_addr` |

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

# macOS arm64
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 \
  go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc_darwin_arm64 ./cmd/frpc/

# 服务端 frps（任意平台）
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -tags "frps,noweb" -trimpath -ldflags="-s -w" -o bin/frps_linux_amd64 ./cmd/frps/
```

---

## 4. 配置示例

### 4.1 客户端 `frpc.toml`

```toml
serverAddr = "your.frps.example.com"   # 也可填 IP4P 编码的 IPv6 地址
serverPort = 7000

# 鉴权令牌，需与服务端一致
auth.token = "请把这里换成你自己的强令牌"

[[proxies]]
name = "ssh"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000
```

启动：`frpc.exe -c frpc.toml`

### 4.2 服务端 `frps.toml`（用官方 v0.72.0 的 frps）

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
- 使用、修改、再分发请遵守 Apache 2.0 许可证，并保留相应版权与许可声明。

---

## 7. 免责声明

本软件仅供在**你拥有或获得明确授权**的设备与网络上进行内网穿透 / 远程访问使用。
使用者须自行遵守所在地法律法规，因滥用本软件造成的任何后果由使用者自行承担。
