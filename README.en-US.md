# 🚀 crypto-monitor: High-Throughput EVM Chain Asset Scanning Engine

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

**crypto-monitor** is a high-performance, anti-blocking command-line (CLI) asset aggregation scanning tool built with Go. It is specifically designed for Web3 data analysts, airdrop hunters, and large address trackers, capable of performing large-scale address balance snapshots across multiple chains and tokens with minimal memory usage and high throughput.

> 💡 **Design Intent:** To address the common challenges faced by traditional Web3 scripts when scanning hundreds of thousands of addresses, such as RPC rate limit blocking (HTTP 429), memory overflow (OOM), multi-chain asset coverage chaos, and program crashes resulting in data loss.

---

## ✨ Core Advanced Features

- ⚡ **CSP Three-Stage Concurrent Pipeline with Backpressure Control**
    - Move away from the traditional single-thread blocking model and establish a three-stage pipeline: network requests → data cleaning → asynchronous persistence.
    - Introduce a channel with a 5000-item buffer capacity, achieving perfect memory isolation and reverse backpressure control, maximizing single-machine performance while preventing **OOM**.
- 🛡️ **Microsecond-Level Smooth Rate Limiting and RPC Aggregation (Multicall3)**
    - Seamlessly integrate the Multicall3 smart contract, bundling hundreds of RPC requests, reducing network I/O overhead by **90%**.
    - Built-in global rate limiter based on the "token bucket algorithm" to precisely control requests per second (RPS) and burst traffic, effectively preventing RPC node IP ban mechanisms.
- 💾 **Enterprise-Level Cold and Hot Data Dual-Write Architecture (SQLite + CSV)**
    - **Zero CGO Dependency**: Use a pure Go SQLite driver (glebarez/sqlite) for cross-platform one-click compilation, eliminating the need for complex GCC environment setup.
    - **Dual Table State Separation**: Maintain AssetRecord (hot data status table) and AssetHistoryLog (cold data log table) in real-time.
    - **Absolute Consistency**: Use local transactions to wrap DB dual-write, combined with ON CONFLICT, for high-performance batch Upsert of a single network interaction.
- 🧬 **Web3 Data Defense Mechanisms (Defensive Programming)**
    - **Multi-chain Anti-Coverage**: The database uses a composite unique index of WalletAddress + TokenContract + ChainID at the underlying level.
    - **Anti-Dirty Read Snapshot**: Implement a front-end BlockHeight timestamp snapshot, tightly bound to batch data, to completely eliminate dirty reads caused by network latency and concurrent disorder.
    - **Dirty Data Interception**: Strictly validate the Success flag, actively downgrade and discard single abnormal data to prevent "False Zero" pollution of the status library.
- 🎨 **Modern CLI and Graceful Shutdown**
    - Adopt a dual-trigger mechanism (100-item threshold + 2-second Ticker) to prevent tail data from remaining in memory.
    - Support multi-path bypass output, with strict rollback of main DB on failure and graceful degradation of bypass CSV on failure.
    - Built-in high-fidelity dynamic progress bar, supporting safe context cancellation and resource recovery.

---

## 📸 Running Effect (Demo)
---
![Image](https://github.com/user-attachments/assets/3829bcb2-9d78-487c-aefd-49d07aac4a28)
---

## 📁 Directory Structure Explanation
---
```text
crypto-monitor/
├── config/             # Configuration parsing module
│   └── config.yaml     # 👈 Your core configuration file
├── data/
│   ├── addresses/      # 👈 Your target address TXT files go here
│   └── cache/          # Program-generated metadata cache files
├── internal/           # Core business logic (private package)
│   ├── engine/         # Concurrent scheduling, task lifecycle, and core workflow engine
│   └── provider/       # Chain interaction layer: RPC client, ABI encoding/decoding, Multicall aggregation, DB service
├── output/             # 👈 Program-generated CSV results are stored here
├── pkg/                # Public basic components
│   ├── metadata/       # Token metadata multi-level cache mechanism
│   └── retry/          # Network request fault tolerance and backoff retry algorithm
├── tools/              # Auxiliary tool classes (data streaming export, etc.)
├── .env                # 👈 Your node private key is stored here (anti-leakage, needs manual creation)
├── main_monitor.db   # 🤖 SQLite local strong consistency database automatically generated based on config.yaml's watchlists name field
├── go.mod / go.sum     # Go module dependency management
└── main.go             # Program main entry point
```
---

## 🛠️ Quick Start
---
### 1. Installation and Compilation
```bash
git clone https://github.com/zz-gopher/crypto-monitor.git
cd crypto-monitor
go build -o crypto-monitor main.go
```
### 2. Local Environment Configuration (.env File)
⚠️ Important Notice: Please manually create a file named .env in the project root directory and copy the following content into it. This file has been ignored by git and is specifically used to store your private RPC links and local network settings.
```bash
# ====== Scientific Internet/Proxy Configuration ======
# If you experience timeout issues with domestic direct-connected RPC nodes, please configure your local proxy port
HTTP_PROXY=http://127.0.0.1:7890
HTTPS_PROXY=http://127.0.0.1:7890
NO_PROXY=localhost,127.0.0.1

# ====== Node Key Configuration ======
# Please apply for your own RPC WSS/HTTPS links from platforms like Alchemy, Infura, or QuickNode for free
ARB_RPC_URL=https://arb-mainnet.g.alchemy.com/v2/你的密钥
ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/你的密钥
```

- **Principle Explanation**: The ${ETH_RPC_URL} variable in config.yaml automatically reads the values from this .env file, achieving separation of code and private configuration.
### 3. Core Engine Configuration (config.yaml)
🔹 App Engine Tuning (Performance and Defense)
```yaml
app:
  retry:
    max_retries: 5       # Fault tolerance: Automatically retry up to 5 times in case of network fluctuations or node errors
    base_delay: 2s       # First retry wait time: 2 seconds
    max_delay: 60s       # Backoff algorithm: Gradually increase retry intervals, up to a maximum of 60 seconds

  global_timeout: 5m     # Safety timeout: Forcefully terminate the entire scanning task after 5 minutes to prevent program deadlock
  timeout: 8s            # Single network timeout: If an RPC request does not respond within 8 seconds, it is considered failed and retried

  concurrency: 20        # Number of workers: Simultaneously dispatch 20 workers to request data from nodes. Too many can cause memory overflow or node banning, too few can result in slow speed
  batch_size: 500        # Batch size: Each worker takes 500 addresses to request data from nodes at once (using Multicall technology to greatly save network requests)

  rate_limit:
    rps: 15              # Token bucket rate: Globally limit RPC node requests to a maximum of 15 per second to protect free node quotas
    burst: 30            # Burst capacity: Allow up to 30 concurrent requests instantaneously to handle sudden network smoothness

  metadata_cache:
    dir: "./data/cache"  # Local cache directory: Store queried token names and precision here; next time scanning the same token will read from disk, eliminating network overhead
    ttl: 720h            # Cache validity: 30 days; token basic information will be re-verified on-chain after 30 days
```
🔹 Output Configuration
```yaml
output:
  csv:
    enabled: true        # Enable CSV output: When true, the screen is silent and all data is written to CSV
    dir: "./output"      # Output directory: Generated CSV files will be stored in this folder
    mode: append         # Write mode: Append mode continues writing to the file end after unexpected program crashes and restarts, without clearing historical data
```
🔹 Networks & Tokens Configuration
```yaml
networks:
  # Define the chains you want to scan; chain_id must be accurate, rpc will automatically read variables from .env file
  ethereum-mainnet: { chain_id: 1, rpc: ["${ETH_RPC_URL}"], native_symbol: ETH}
  arbitrum-one:     { chain_id: 42161, rpc: ["${ARB_RPC_URL}"], native_symbol: ETH}

tokens:
  # Define ERC20 and ERC721 tokens you need to monitor (native gas coins don't need configuration), and their real contract addresses on different chains
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
🔹 Watchlists Task Configuration
```yaml
watchlists:
  - name: main-monitor                          # Task name: The final report file will be named main-monitor_results.csv
    networks: [ethereum-mainnet, arbitrum-one]  # Task scope: Simultaneously scan Ethereum mainnet and ARB chain
    address_glob: "./data/addresses/*.txt"      # Data source: Automatically read wallet addresses from .txt files in this directory
    assets:
      - token: native                           # Query native Gas coin (ETH)
      - token: USDT                             # Query USDT (will automatically find contract address from the tokens configuration)
      - token: USDC                             # Query USDC
```
- ⚠️ Data Source Preparation Guide (address_glob):
  During program execution, it will strictly search for target wallet addresses according to the specified address_glob path. Please ensure that the folder is created and address files are placed in advance.

- File storage: According to the example configuration, you need to manually create the ./data/addresses/ folder and place your .txt files in it.

- Format requirements: Strictly follow the "one address per line" rule. Addresses must be standard EVM addresses (starting with 0x), without trailing commas, semicolons, or extra spaces and blank lines.

✅ Correct .txt file content example:
``` text
0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045
0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984
0x28C6c06298d514Db089934071355E5743bf21d60
```
### 4. One-click Execution
``` bash
go run main.go -config ./config/config.yaml
```

---

## 🏗️ Architecture Overview (Pipeline Architecture)

The project adopts the modern stream data processing standard **three-stage CSP pipeline** design:

### 1. Producer Layer (Network/RPC)
* Token bucket rate limiter intercepts excessive traffic.
* Obtain the current chain's `BlockHeight` as a consistency snapshot.
* Multicall3 batch retrieves balances, validates `Success` flag, and sends `QueryResult` through Channel.

### 2. Intermediate Layer (UI & Transformer)
* Main goroutine receives results, drives terminal high-fidelity progress bar (`progressbar`).
* Filters dirty data, converts underlying structures to DB entity model `AssetRecord`.
* Pushes data into a `dataChan` with 5000 buffer capacity, utilizing blocking mechanism to achieve reverse backpressure.

### 3. Consumer Layer (Storage Daemon)
* Independent background storage daemon listens to `dataChan`.
* Triggers persistence logic when 100 items are full or after 2 seconds (Ticker).
* Opens local `Transaction`, executes status table Upsert and AssetHistoryLog Insert.
* After DB persistence is successful, use `WriteBatch` to append data to CSV bypass with zero lock overhead.

---

## 🤝 Contribution and License
This project is licensed under the [MIT License](LICENSE). Contributions are welcome to build the world's most powerful scanning engine!
