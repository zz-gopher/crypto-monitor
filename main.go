package main

import (
	"context"
	"crypto-monitor/config"
	"crypto-monitor/internal/engine"
	"crypto-monitor/internal/provider"
	"crypto-monitor/internal/provider/eth/contracts/multicall3"
	"crypto-monitor/tools"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
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
	// 默认总超时设定为30秒
	ctxAll, cancelAll := context.WithTimeout(context.Background(), 30*time.Second)
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

	// 资产查询任务
	for _, wl := range cfg.Watchlists {
		// 读取地址文件
		addresses, err := config.LoadAddressesFromTXT(wl.AddressGlob)
		if err != nil {
			log.Fatalf("读取地址文件失败: %v", err)
		}
		// 打印地址数量和前 3 个地址
		fmt.Printf("加载了 %d 个地址:\n", len(addresses))
		// 地址进行切片
		batches := tools.SplitAddresses(addresses, cfg.App.BatchSize)
		// 创建用于接收结果的 Channel
		resultsChan := make(chan QueryResult, len(wl.Networks)*len(wl.Assets)*len(batches))
		var wg sync.WaitGroup
		for _, ass := range wl.Assets {
			for _, network := range wl.Networks {
				wg.Add(1)
				// 开启协程
				go func(n string, asset config.AssetRef) {
					defer wg.Done()
					runtime, ok := runtimes[n]
					if !ok || runtime == nil {
						fmt.Printf("⚠️ 初始化 %s 网络失败或不存在,跳过\n", n)
						return
					}
					if asset.Token == multicall3.AssetTypeNative {
						// 每一个批次理论上会走一次RPC,减缓了RPC的压力
						for _, batch := range batches {
							tokenBalances, err := runtime.MultiChecker.CheckToken(multicall3.AssetTypeNative,
								common.Address{}, // 自动等价于 0x000...
								ctxAll, cfg.App.Timeout, batch)
							resultsChan <- QueryResult{Network: n, Token: runtime.NativeSymbol, Balances: tokenBalances, Error: err}
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
						tokenBalances, err := runtime.MultiChecker.CheckToken(
							tokenCfg.Type,
							common.HexToAddress(onNetwork.Contract),
							ctxAll, cfg.App.Timeout, batch)
						resultsChan <- QueryResult{Network: n, Token: asset.Token, Balances: tokenBalances, Error: err}
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
		results := make(map[string]map[string][]provider.TokenBalance)
		var successCount int
		for res := range resultsChan {
			if res.Error != nil {
				fmt.Printf("❌ 网络 %s 读取 %s 失败: %v\n", res.Network, res.Token, res.Error)
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
				if b.Success {
					fmt.Printf("✅ [%s] 地址: %s | 余额: %s %s\n",
						res.Network, tools.ShortAddress(b.Owner), b.Balance.String(), res.Token)
					successCount++
				}
			}
		}
		totalExpected := len(addresses) * len(wl.Networks) * len(wl.Assets)
		fmt.Printf("\n--------------------------------------------------\n")
		fmt.Printf("📊 Summary Report\n")
		fmt.Printf("--------------------------------------------------\n")
		fmt.Printf("✅ Success Rate : %d / %d\n", successCount, totalExpected)
		fmt.Printf("🎉 All tasks completed! Time: %v\n", time.Since(startTime))
		fmt.Printf("--------------------------------------------------\n")
	}

}
