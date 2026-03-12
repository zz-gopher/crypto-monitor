package main

import (
	"context"
	"crypto-monitor/config"
	"crypto-monitor/internal/engine"
	"crypto-monitor/internal/provider"
	"crypto-monitor/internal/provider/eth/contracts/multicall3"
	"crypto-monitor/pkg/retry"
	"crypto-monitor/tools"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/time/rate"
)

// QueryResult 用于在 Channel 中传递结果
type QueryResult struct {
	Network  string
	Token    string
	Balances []provider.TokenBalance
	Error    error
}

func main() {
	// go run . --config ./config.yaml
	cfgPath := flag.String("config", "./config/config.yaml", "path to config yaml")
	flag.Parse()
	// 设置代理
	//_ = os.Setenv("HTTP_PROXY", "http://127.0.0.1:7890")
	//_ = os.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
	// 读取配置文件
	// 加载 .env（可选：不存在就忽略）
	if err := godotenv.Load(); err != nil {
		log.Printf("warn: .env not loaded: %v (ok if you set env vars another way)", err)
	}
	cfg, err := config.LoadCfg(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}
	if len(cfg.Networks) == 0 {
		log.Fatalf("配置文件没有网络列表:")
	}
	startTime := time.Now()
	// 默认总超时设定
	ctxAll, cancelAll := context.WithTimeout(context.Background(), cfg.App.GlobalTimeout)
	defer cancelAll()
	runtimes, failed, err := engine.InitNetworks(ctxAll, cfg, cfg.App.Timeout)

	if err != nil {
		log.Fatalf("初始化网络连接失败: %v:", err)
	}
	if len(failed) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "⚠️ 部分网络初始化失败（将跳过这些网络）：\n")
		for name, e := range failed {
			_, _ = fmt.Fprintf(os.Stderr, "   - %s: %v\n", name, e)
		}
	}
	fmt.Printf("✅ 初始化成功网络数量: %d\n", len(runtimes))
	for name, rt := range runtimes {
		fmt.Printf("   - %s (chain_id=%d, rpc=%s, native=%s)\n", name, rt.ChainID, rt.RPCUsed, rt.NativeSymbol)
	}

	// 初始化全局令牌桶 (限制绝对速率 RPS，保护远端节点)
	limiter := rate.NewLimiter(rate.Limit(cfg.App.RateLimit.RPS), cfg.App.RateLimit.Burst)

	// 信号量 (Semaphore)：控制本地的最大并发协程数，防止内存溢出和句柄耗尽
	sem := make(chan struct{}, cfg.App.Concurrency)

	var globalSuccessCount int
	var globalTotalExpected int

	// 根据配置文件，查询资产任务
	for _, wl := range cfg.Watchlists {

		// 为当前 Watchlist 准备独立的 CSV
		wlConfig := cfg.Output.CSV
		wlPath := filepath.Join(wlConfig.Dir, fmt.Sprintf("%s_results.csv", wl.Name))
		wlExporter, err := tools.NewCSVExporter(wlConfig, wlPath)
		if err != nil {
			fmt.Printf("❌ 无法为 %s 创建 Exporter: %v\n", wl.Name, err)
			continue
		}
		// 读取地址文件
		addresses, err := config.LoadAddressesFromTXT(wl.AddressGlob)
		if err != nil {
			log.Fatalf("读取地址文件失败: %v", err)
		}
		// 地址进行切片
		batches := tools.SplitAddresses(addresses, cfg.App.BatchSize)

		// 计算当前 Watchlist 的预期总量并累加到全局
		currentWlExpected := len(addresses) * len(wl.Networks) * len(wl.Assets)
		globalTotalExpected += currentWlExpected

		fmt.Printf("\n🚀 正在处理任务 [%s] | 地址总数: %d | 预期结果: %d\n", wl.Name, len(addresses), currentWlExpected)

		// 创建用于接收结果的 Channel
		resultsChan := make(chan QueryResult, len(wl.Networks)*len(wl.Assets)*len(batches))

		var wg sync.WaitGroup

		// ------------开启并发查询------------
		for _, ass := range wl.Assets {
			for _, network := range wl.Networks {
				wg.Add(1)
				// 开启协程
				sem <- struct{}{} // 获取信号
				go func(n string, asset config.AssetRef) {
					defer wg.Done()
					defer func() { <-sem }() // 协程结束后释放信号
					runtime, ok := runtimes[n]
					if !ok || runtime == nil {
						fmt.Printf("⚠️ 初始化 %s 网络失败或不存在,跳过\n", n)
						return
					}
					if asset.Token == multicall3.AssetTypeNative {
						// 每一个批次理论上会走一次RPC,减缓了RPC的压力
						for _, batch := range batches {
							var tokenBalances []provider.TokenBalance
							retryErr := retry.Do(ctxAll, cfg.App.Retry, func() error {
								if err := limiter.Wait(ctxAll); err != nil {
									return fmt.Errorf("令牌桶排队被打断或超时: %w", err)
								}
								// 拿到令牌
								var err error
								tokenBalances, err = runtime.MultiChecker.CheckToken(multicall3.AssetTypeNative,
									common.Address{}, // 自动等价于 0x000...
									ctxAll, cfg.App.Timeout, batch)

								return err
							})
							resultsChan <- QueryResult{Network: n, Token: runtime.NativeSymbol, Balances: tokenBalances, Error: retryErr}
						}
						return
					}

					// 处理其他类型的 Token
					tokenCfg, ok := cfg.Tokens[asset.Token]
					if !ok {
						fmt.Printf("⚠️ 未找到代币 %s 的配置\n", asset.Token)
						return
					}
					onNetwork, ok := tokenCfg.PerNetwork[n]
					if !ok {
						// 某些网络可能没有该代币合约，正常跳过，不需要报错
						return
					}
					for _, batch := range batches {
						var tokenBalances []provider.TokenBalance
						retryError := retry.Do(ctxAll, cfg.App.Retry, func() error {
							if err := limiter.Wait(ctxAll); err != nil {
								return fmt.Errorf("令牌桶排队被打断或超时: %w", err)
							}
							// 拿到令牌
							var err error
							tokenBalances, err = runtime.MultiChecker.CheckToken(
								tokenCfg.Type,
								common.HexToAddress(onNetwork.Contract),
								ctxAll, cfg.App.Timeout, batch)
							return err
						})
						resultsChan <- QueryResult{Network: n, Token: asset.Token, Balances: tokenBalances, Error: retryError}
					}
				}(network, ass)
			}
		}
		// 另启一个线程等待所有任务完成，然后关闭channel
		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// 主协程统一从 Channel 收集结果并组装 Map
		var wlSuccessCount int
		results := make(map[string]map[string][]provider.TokenBalance)
		// 进度条初始化，只在(写CSV)时用它
		var bar *progressbar.ProgressBar
		if cfg.Output.CSV.Enabled {
			// Default 参数：总量，前缀提示词
			bar = progressbar.Default(int64(currentWlExpected), fmt.Sprintf("🚀 [%s] 扫描中", wl.Name))
		}
		for res := range resultsChan {
			if res.Error != nil {
				fmt.Printf("❌ [%s] 读取 %s 失败: %v\n", res.Network, res.Token, res.Error)
				continue
			}

			// 懒加载初始化 Map
			if results[res.Network] == nil {
				results[res.Network] = make(map[string][]provider.TokenBalance)
			}
			// 追加数据
			results[res.Network][res.Token] = append(results[res.Network][res.Token], res.Balances...)

			// 边接收边统计，节省一次后续的遍历
			for _, b := range res.Balances {
				row := []string{
					res.Network,
					b.TokenAddress.Hex(),
					b.Owner.Hex(),
					b.Symbol,
					b.Balance.String(),
					fmt.Sprintf("%d", b.Decimals),
					fmt.Sprintf("%t", b.Success),
				}
				if cfg.Output.CSV.Enabled {
					// 写入CSV 并展示控制台
					_ = wlExporter.WriteRow(row)
					// 进度条 +1
					if bar != nil {
						_ = bar.Add(1)
					}
				} else {
					// 不写文件，全量打印控制台
					if b.Success {
						fmt.Printf("✅ [%s] 地址: %s | 余额: %s %s\n",
							res.Network, tools.ShortAddress(b.Owner), b.Balance.String(), res.Token)
					} else {
						// 如果失败了，控制台给个提示
						fmt.Printf("⚠️  [%s] 地址: %s 代币:%s 查询失败\n", res.Network, res.Token, tools.ShortAddress(b.Owner))
					}
				}
				if b.Success {
					wlSuccessCount++
				}
			}
		}
		// 关闭当前 CSV 文件句柄（确保 Flush 到磁盘）
		wlExporter.Close()

		// 累加到全局统计
		globalSuccessCount += wlSuccessCount

		fmt.Printf("✅ 任务 [%s] 完成! 成功率: %d/%d\n", wl.Name, wlSuccessCount, currentWlExpected)
	}
	fmt.Printf("\n==================================================\n")
	fmt.Printf("📊 ALL TASKS COMPLETED (全部查询完毕)\n")
	fmt.Printf("==================================================\n")
	fmt.Printf("⏱️ 总耗时     : %s\n", time.Since(startTime))
	fmt.Printf("📈 总成功率   : %d / %d (%.2f%%)\n",
		globalSuccessCount,
		globalTotalExpected,
		float64(globalSuccessCount)/float64(globalTotalExpected)*100)
	if cfg.Output.CSV.Enabled {
		fmt.Printf("📂 数据报告已分别保存至 ./output/ 目录下\n")
	}
	fmt.Printf("==================================================\n")
}
