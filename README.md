# FRP-IP4P（frpc 内网穿透 · 客户端支持 IP4P 地址）

> 本仓库是基于 [fatedier/frp](https://github.com/fatedier/frp) **v0.72.0** 的一个轻量二次开发分支。
> 仅在**客户端（frpc）** 上新增了 **IP4P 地址解析** 支持，其余功能与官方 v0.72.0 完全一致。
> 服务端（frps）**无需任何改动**，可直接使用官方 v0.72.0 的 frps。

本分支**没有**对二进制做任何“反检测 / 免杀”改造，成品即标准 frp 客户端，便于审计与自行编译。

## 什么是 IP4P

当 frps 位于 NAT 之后、公网 IP 与端口会动态变化时，frpc 需要在连接前查出真实的 `IP:端口`。
本分支让 `serverAddr` 填一个域名即可，按以下顺序自动解析，命中即返回：

| 优先级 | 方式 | 记录内容 |
| --- | --- | --- |
| 1 | **TXT 记录**（natmap / lucky / DDNS 常用） | `base64("IPv4:端口")`，例如 `MjAzLjAuMTEzLjc6MTgyNDY=` 解码得 `203.0.113.7:18246`；也兼容明文 |
| 2 | **AAAA 记录** | IP4P 编码的 IPv6 地址（`2001:0000::/16`，端口在字节 10-11、IPv4 在字节 12-15） |
| 3 | 普通模式 | 无匹配记录时原样使用 `serverAddr` + `serverPort`，与官方 frp 完全一致 |

**解析成功后记录里的端口会覆盖 `serverPort`**，所以使用 IP4P 时可不写 `serverPort`。
日志会打印解析结果，例如 `ip4p: resolved server address from TXT record, xxx -> 203.0.113.7:18246`。

## 与原版 frp 的差异（仅客户端）

| 文件 | 改动 |
| --- | --- |
| `client/ip4p.go` | 新增：`lookupIP4P(ctx, addr, port)`，TXT + AAAA 两级解析，失败回退普通模式 |
| `client/connector.go` | QUIC 拨号、TCP/WS 拨号两处各插入一次 `lookupIP4P` 解析 `server_addr` |
| `client/ip4p_test.go` | 新增：TXT 解析与回退行为的单元测试 |

## 编译（从源码）

需要 Go **1.24+**（本分支在 Go 1.26 下验证通过）。

```powershell
# Windows x64 客户端
$env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc.exe ./cmd/frpc/
```

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc_linux_amd64 ./cmd/frpc/

# macOS arm64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
  go build -tags "frpc,noweb" -trimpath -ldflags="-s -w" -o bin/frpc_darwin_arm64 ./cmd/frpc/
```

> 标签 `noweb` 用于跳过内置 Web 资源的嵌入；如需带 Web 管理页面，请先按官方流程 `make web` 生成 `web/frpc/dist`。

## 快速配置

使用 IP4P（TXT 模式）的 `frpc.toml`：

```toml
serverAddr = "your.ip4p.example.com"   # TXT 记录存 base64("公网IP:端口")
# serverPort 可不写：解析出的端口会自动覆盖默认值

auth.method = "token"
auth.token  = "请把这里换成你自己的强令牌"
# dnsServer = "192.168.1.1:53"        # 可选：指定 DNS（TXT 查询也会走它）

[[proxies]]
name      = "rdp"
type      = "stcp"
secretKey = "请把这里换成你自己的密钥"
localIP   = "127.0.0.1"
localPort = 3389
```

启动：`frpc.exe -c frpc.toml`

> 排查：用 `nslookup -type=TXT your.ip4p.example.com` 确认 TXT 记录存在且未被 DNS 服务器过滤；
> 用 `base64 -d` 解码记录值，应得到 `IP:端口`。无 IP4P 记录时按普通模式连接，无需额外配置。

## 下载（预编译二进制）

前往 [Releases](https://github.com/wengjing086/frp-ip4p/releases) 下载，包含 17 个平台的 frpc：

| 平台 | 资产文件名 |
| --- | --- |
| Windows x64 / ARM64 / x86 | `frpc_windows_amd64.exe`（也提供兼容名 `frpc.exe`）、`frpc_windows_arm64.exe`、`frpc_windows_386.exe` |
| Linux x86_64 / ARM64 / ARMv7 / x86 | `frpc_linux_amd64`、`frpc_linux_arm64`、`frpc_linux_arm`、`frpc_linux_386` |
| Linux MIPS（路由器，含大端/小端） | `frpc_linux_mipsle`、`frpc_linux_mips`、`frpc_linux_mips64le`、`frpc_linux_mips64` |
| Linux RISC-V 64 / 龙芯 LoongArch64 | `frpc_linux_riscv64`、`frpc_linux_loong64` |
| macOS Intel / Apple Silicon | `frpc_darwin_amd64`、`frpc_darwin_arm64` |
| FreeBSD x86_64 / ARM64 | `frpc_freebsd_amd64`、`frpc_freebsd_arm64` |
| 校验和 | `SHA256SUMS.txt` |

Linux / macOS / FreeBSD 下载后需 `chmod +x` 再运行；校验用 `sha256sum -c SHA256SUMS.txt`。
服务端 frps 无需改动，请直接使用官方 v0.72.0 对应平台的 frps。

## 详细说明

完整的 IP4P 原理、编译、配置、以及**杀毒软件误报的合规处理**、许可证与免责声明，见 [IP4P.md](IP4P.md)。

## 许可证与免责声明

- 整体基于 fatedier/frp v0.72.0，遵循 **Apache License 2.0**；IP4P 实现移植自 [hevz/frp](https://github.com/hevz/frp)（ip4p 分支，Apache 2.0），TXT 解析参考 [qaz741wsd856/frp](https://github.com/qaz741wsd856/frp)。
- 本软件仅供在**你拥有或获得明确授权**的设备与网络上使用，使用者须自行遵守所在地法律法规，因滥用造成的后果由使用者自行承担。
