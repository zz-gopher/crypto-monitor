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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

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
		// 按网络、资产归档的余额查询结果
		results := make(map[string]map[string][]provider.TokenBalance)
		for _, t := range wl.Assets {
			for _, network := range wl.Networks {
				runtime := runtimes[network]
				if runtime == nil {
					fmt.Printf("⚠️ 初始化%s网络失败,跳过: \n", network)
					continue
				}
				batches := tools.SplitAddresses(addresses, cfg.Networks[runtime.Name].BatchSize)
				// 地址切片
				if t.Token == multicall3.AssetTypeNative {
					for _, batch := range batches {
						tokenBalances, err := runtime.MultiChecker.CheckToken(multicall3.AssetTypeNative,
							common.HexToAddress("0x0000000000000000000000000000000000000000"),
							ctxAll, cfg.App.Timeout, batch)
						if err != nil {
							log.Fatalf("网络%smulticall合约读取%s余额失败: %v", runtime.Name, runtime.NativeSymbol, err)
						}
						if _, ok := results[runtime.Name]; !ok {
							results[runtime.Name] = make(map[string][]provider.TokenBalance)
						}

						results[runtime.Name][runtime.NativeSymbol] = append(results[runtime.Name][runtime.NativeSymbol], tokenBalances...)
					}
				} else {
					tokenCfg, ok := cfg.Tokens[t.Token]
					if !ok {
						fmt.Printf("⚠️ 未找到代币 %s 的配置，跳过\n", t.Token)
						continue
					}
					onNetwork, ok := tokenCfg.PerNetwork[network]
					if !ok {
						fmt.Printf("⚠️ 代币 %s 未配置网络 %s，跳过\n", t.Token, network)
						continue
					}
					for _, batch := range batches {
						tokenBalances, err := runtime.MultiChecker.CheckToken(tokenCfg.Type, common.HexToAddress(onNetwork.Contract), ctxAll, cfg.App.Timeout, batch)
						if err != nil {
							log.Fatalf("网络%smulticall合约读取%s余额失败: %v", runtime.Name, t.Token, err)
						}
						if _, ok := results[runtime.Name]; !ok {
							results[runtime.Name] = make(map[string][]provider.TokenBalance)
						}
						results[runtime.Name][t.Token] = append(results[runtime.Name][t.Token], tokenBalances...)
					}
				}
			}
		}
		// 控制台打印
		for network, assetMap := range results {
			for token, balances := range assetMap {
				for _, b := range balances {
					fmt.Printf("✅ [%s] 地址: %s | 余额: %s %s\n",
						network,
						tools.ShortAddress(b.Owner),
						b.Balance.String(),
						token,
					)
				}
			}
		}
	}
}
