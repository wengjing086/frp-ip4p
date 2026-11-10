# FRP-IP4P（frpc 内网穿透 · 客户端支持 IP4P 地址）

> 本仓库是基于 [fatedier/frp](https://github.com/fatedier/frp) **v0.72.0** 的一个轻量二次开发分支。
> 仅在**客户端（frpc）** 上新增了 **IP4P 地址解析** 支持，其余功能与官方 v0.72.0 完全一致。
> 服务端（frps）**无需任何改动**，可直接使用官方 v0.72.0 的 frps。

本分支**没有**对二进制做任何“反检测 / 免杀”改造，成品即标准 frp 客户端，便于审计与自行编译。

## 什么是 IP4P

IP4P（IP for P2P）把 **IPv4 地址 + 端口** 编码进 **IPv6 地址**（前缀 `2001:0000::/16`），用于 NAT 穿透 / P2P 场景。
在本分支中，`server_addr` 既可填普通域名 / IPv4，也可填一个 **IP4P 编码的 IPv6 地址**；
客户端在发起连接（TCP / WebSocket / QUIC）前会自动把它解码回真实 IPv4 + 端口。

```
2001:0000::IPv4:PORT  ──解码──▶  IPv4 地址 + 端口
```

如果 `server_addr` 不是 IP4P 地址，则原样连接，行为与普通 frp 完全相同。

## 与原版 frp 的差异（仅客户端）

| 文件 | 改动 |
| --- | --- |
| `client/ip4p.go` | 新增：`lookupIP4P(addr, port)` 解码 IP4P 地址 |
| `client/connector.go` | QUIC 拨号、TCP/WS 拨号两处各插入一次 `lookupIP4P` 解析 `server_addr` |

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

`frpc.toml`：

```toml
serverAddr = "your.frps.example.com"   # 也可填 IP4P 编码的 IPv6 地址
serverPort = 7000
auth.token = "请把这里换成你自己的强令牌"

[[proxies]]
name = "ssh"
type = "tcp"
localIP = "127.0.0.1"
localPort = 22
remotePort = 6000
```

启动：`frpc.exe -c frpc.toml`

## 详细说明

完整的 IP4P 原理、编译、配置、以及**杀毒软件误报的合规处理**、许可证与免责声明，见 [IP4P.md](IP4P.md)。

## 许可证与免责声明

- 整体基于 fatedier/frp v0.72.0，遵循 **Apache License 2.0**；IP4P 实现移植自 [hevz/frp](https://github.com/hevz/frp)（ip4p 分支，Apache 2.0）。
- 本软件仅供在**你拥有或获得明确授权**的设备与网络上使用，使用者须自行遵守所在地法律法规，因滥用造成的后果由使用者自行承担。
