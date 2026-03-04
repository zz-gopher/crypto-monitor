package main

import (
	"context"
	"crypto-monitor/config"
	"crypto-monitor/internal/engine"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// go run . --config ./config.yaml
	cfgPath := flag.String("config", "./config/config.yaml", "path to config yaml")
	flag.Parse()

	// 读取配置文件
	cfg, err := config.LoadCfg(*cfgPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	if len(cfg.Networks) == 0 {
		log.Fatalf("load config has no networks:")
	}
	// 读取地址文件
	addresses, err := config.LoadAddressesFromTXT("./data/addresses/*.txt")
	if err != nil {
		log.Fatalf("读取地址文件失败: %v", err)
	}
	// 打印地址数量和前 3 个地址
	fmt.Printf("加载了 %d 个地址:\n", len(addresses))

	// 默认总超时设定为30秒
	ctxAll, cancelAll := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelAll()
	runtimes, failed, err := engine.InitNetworks(ctxAll, cfg, cfg.App.Timeout)

	if err != nil {
		log.Fatalf("InitNetworks error: %v:", err)
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

}
