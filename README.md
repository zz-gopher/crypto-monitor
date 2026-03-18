# 🚀 EVM-Asset-Scanner: 工业级 EVM 链上资产高并发扫描引擎

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

**EVM-Asset-Scanner** 是一个基于 Go 语言开发的高性能、防封禁的命令行 (CLI) 资产聚合扫描工具。它专为 Web3 数据分析师、空投猎人及大户地址追踪设计，能够在极低的内存占用下，以极高的吞吐量完成多链、多代币的海量地址余额快照。

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
![Image](https://github.com/user-attachments/assets/3829bcb2-9d78-487c-aefd-49d07aac4a28)