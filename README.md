# 🚀 crypto-monitor: EVM 链上资产高并发扫描引擎

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

**crypto-monitor** 是一个基于 Go 语言开发的高性能、防封禁的命令行 (CLI) 资产聚合扫描工具。它专为 Web3 数据分析师、空投猎人及大户地址追踪设计，能够在极低的内存占用下，以极高的吞吐量完成多链、多代币的海量地址余额快照。

> 💡 **设计初衷：** 解决传统 Web3 脚本在面对十万级地址扫描时经常遇到的 **RPC 限流封禁 (HTTP 429)**、**内存溢出 (OOM)** 以及 **上古非标合约解析崩溃** 等痛点。

---

## ✨ 核心硬核特性 (Features)

- ⚡ **极致并发与 RPC 节流 (Multicall3)**
    - 基于协程池与 Channel 构筑生产消费模型。
    - 深度集成 Multicall3 智能合约，将成百上千个散碎的 RPC 请求打包聚合，**网络 I/O 开销降低 90%**。
- 🛡️ **微秒级平滑限流 (Token Bucket Rate Limiting)**
    - 内置基于“令牌桶算法”的全局限流器，精准控制每秒请求数 (RPS) 与突发流量 (Burst)，完美规避免费 RPC 节点的 IP 封禁机制。
- 💾 **防 OOM 的流式数据落盘**
    - 摒弃传统的“全量内存聚合”模式，采用边查边写的流式 CSV 导出引擎。
    - 支持 `append` 断点续写与 `flush_every` 定期刷盘，百万级数据扫描时内存占用始终保持在常数级 **O(1)**。
- 🧬 **双层降级解码 (兼容非标合约)**
    - 底层重写 `go-ethereum` 的 ABI 解包逻辑。
    - 独创 `string / bytes32` 降级解码器，完美兼容 MakerDAO (MKR) 等 2017 年远古非标 ERC-20 合约，实现全链代币 100% 容错解析。
- 🎨 **现代化 CLI 极客体验**
    - 提供 测试 / 跑批 双模式。跑批模式下实施绝对静默策略 (Silent Mode)，彻底杜绝高并发写入时的终端花屏撕裂，仅保留高保真动态进度条 (ETA) 实时掌控全局扫描进度。。

---

## 📸 运行效果 (Demo)
---
![Image](https://github.com/user-attachments/assets/3829bcb2-9d78-487c-aefd-49d07aac4a28)
---

## 📁 目录结构说明
---
```text
crypto-monitor/
├── config/             # 配置解析模块
│   └── config.yaml     # 👈 你的核心配置文件
├── data/
│   ├── addresses/      # 👈 你的目标地址 TXT 文件放这里
│   └── cache/          # 程序自动生成的元数据缓存文件
├── internal/           # 核心业务逻辑 (私有包)
│   ├── engine/         # 并发调度、任务生命周期与核心工作流引擎
│   └── provider/       # 链上交互层：RPC 客户端、ABI 编解码与 Multicall 聚合
├── output/             # 👈 程序自动生成的 CSV 结果存这里
├── pkg/                # 公共基础组件
│   ├── metadata/       # Token 元数据多级缓存机制
│   └── retry/          # 网络请求容错与退避重试算法
├── tools/              # 辅助工具类 (数据流式导出等)
├── .env                # 👈 你的节点私钥放这里 (防泄露，需手动创建)
├── go.mod / go.sum     # Go 模块依赖管理
└── main.go             # 程序主入口
```
---

## 🛠️ 快速开始 (Quick Start)
---
### 1. 安装与编译
```bash
git clone https://github.com/zz-gopher/crypto-monitor.git
cd crypto-monitor
go build -o crypto-monitor main.go
```
### 2. 本地环境配置 (.env 文件)
⚠️ 重要提示： 请在项目根目录下手动创建一个名为 .env 的文件，并将以下内容复制进去。此文件已被 git 忽略，专门用于存放你的私密 RPC 链接和本地网络设置。
```bash
# ====== 科学上网/代理配置 ======
# 如果你在国内直连 RPC 节点超时，请配置你本地的代理端口
HTTP_PROXY=http://127.0.0.1:7890
HTTPS_PROXY=http://127.0.0.1:7890
NO_PROXY=localhost,127.0.0.1

# ====== 节点密钥配置 ======
# 请去 Alchemy, Infura 或 QuickNode 等平台免费申请你自己的 RPC WSS/HTTPS 链接
ARB_RPC_URL=https://arb-mainnet.g.alchemy.com/v2/你的密钥
ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/你的密钥
```

- **原理解释**： 配置文件 config.yaml 中的 ${ETH_RPC_URL} 会自动读取这个 .env 文件里的值，实现了代码与私密配置的分离。
### 3. 核心引擎配置 (config.yaml)
🔹 App 引擎调优 (性能与防御)
```yaml
app:
  retry:
    max_retries: 5       # 容错机制：遇到网络波动或节点报错，最多自动重试 5 次
    base_delay: 2s       # 首次重试等待 2 秒
    max_delay: 60s       # 退避算法：重试间隔逐渐变长，但最长不超过 60 秒

  global_timeout: 5m     # 兜底超时：防止程序假死，整个扫币任务最长运行 5 分钟会被强制掐断
  timeout: 8s            # 单次网络超时：发给 RPC 的单个请求，如果 8 秒没响应就判定失败并重试

  concurrency: 20        # 协程数量：同时派出 20 个“工人”去向节点要数据。太大容易导致内存溢出或被节点封禁，太小则速度慢
  batch_size: 500        # 打包数量：每个工人一次性拿着 500 个地址去问节点（利用 Multicall 技术极大节省网络请求）

  rate_limit:
    rps: 15              # 漏桶/令牌桶速率：全局严格限制每秒最多只能向节点发 15 个请求，保护免费节点额度
    burst: 30            # 突发容量：允许瞬间并发最多 30 个请求，应对网络突然通畅的情况

  metadata_cache:
    dir: "./data/cache"  # 本地缓存库：查过的代币名字、精度会存进这里，下次扫同样的币直接读硬盘，零网络开销
    ttl: 720h            # 缓存保质期：30天。30天后才会重新去链上核对代币基础信息
```
🔹 Output 结果输出
```yaml
output:
  csv:
    enabled: true        # 开关：设为 false 时屏幕疯狂打印结果（适合测试）；设为 true 时屏幕静默，数据全进 CSV（适合跑批）
    dir: "./output"      # 报告存放地：生成的 csv 文件会存放在这个文件夹下
    mode: append         # 写入模式：append 代表追加，即使程序意外崩溃，重启后也会接着文件末尾继续写，不会清空历史数据
    flush_every: 200     # 刷盘频率：每查到 200 条数据就强制存入硬盘一次，防止断电导致内存数据丢失
```
🔹 Networks & Tokens 链与资产字典
```yaml
networks:
  # 定义你要扫哪些链，chain_id 必须准确，rpc 会自动读取 .env 文件里的变量
  ethereum-mainnet: { chain_id: 1, rpc: ["${ETH_RPC_URL}"], native_symbol: ETH}
  arbitrum-one:     { chain_id: 42161, rpc: ["${ARB_RPC_URL}"], native_symbol: ETH }

tokens:
  # 定义你需要关注的 ERC20 代币 和 ERC721 代币（原生gas币不需要配置），以及它们在不同链上的真实合约地址
  USDT:
    type: erc20
    per_network:
      ethereum-mainnet: { contract: "0xdAC17F958D2ee523a2206206994597C13D831ec7" }
      arbitrum-one:     { contract: "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9" }
  USDC:
    type: erc20
    per_network:
      ethereum-mainnet: { contract: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48" }
      arbitrum-one:     { contract: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831" }
```
🔹 Watchlists 任务编排
```yaml
watchlists:
  - name: main-monitor                          # 任务名称：最后生成的报告文件会命名为 main-monitor_results.csv
    networks: [ethereum-mainnet, arbitrum-one]  # 任务范围：同时扫描以太坊主网和 ARB 链
    address_glob: "./data/addresses/*.txt"      # 数据源：自动读取该目录下所有 .txt 文件里的钱包地址
    assets:
      - token: native                           # 查原生 Gas 币 (ETH)
      - token: USDT                             # 查 USDT (会自动去上面的 tokens 字典里找合约地址)
      - token: USDC                             # 查 USDC
```
- ⚠️ 数据源准备指南 (address_glob)：
  程序运行时，会严格按照 address_glob 指定的路径去寻找目标钱包地址。请务必提前建好文件夹并放入地址文件。

- 文件存放：按示例配置，你需要手动创建 ./data/addresses/ 文件夹，并将你的 .txt 文件放入其中。

- 格式要求：必须严格遵守“一行一个地址”的规则。 必须是标准的 EVM 地址（0x 开头），请勿在行尾添加逗号、分号，或夹杂多余的空格与空行。

✅ 正确的 .txt 文件内容示例：
``` text
0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045
0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984
0x28C6c06298d514Db089934071355E5743bf21d60
```
### 4. 一键运行
``` bash
go run main.go -config ./config/config.yaml
```

---

## 🏗️ 架构概览 (Architecture)
1. **配置解析层**: 加载 YAML，初始化多级缓存机制防并发击穿 (`singleflight` + `sync.Map`)。
2. **调度层**: 控制多 Watchlist 的生命周期，初始化专属的流式 CSV Writer 与 CLI 进度条。
3. **并发工作流**:
    - `Semaphore` 控制本地内存的最大协程驻留量。
    - `RateLimiter` 节流远端 RPC 请求。
4. **底层通信**: Multicall3 打包 -> ABI 双层容错解包 -> 数据格式化输出。
---

## 🤝 贡献与许可 (License)
本项目采用 [MIT License](LICENSE) 开源协议。欢迎提交 Pull Request 一起打造地表最强的扫币引擎！