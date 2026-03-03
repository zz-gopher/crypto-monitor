package main

import (
	"crypto-monitor/config"
	"flag"
	"fmt"
	"log"
)

func main() {
	// go run . --config ./config.yaml
	cfgPath := flag.String("config", "./config/config.yaml", "path to config yaml")
	flag.Parse()

	// 读取配置文件
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	if len(cfg.Networks) == 0 {
		fmt.Errorf("aggregate3 返回数量不一致")
	}
	fmt.Println("PollInterval:", cfg.App.PollInterval)
	fmt.Println("Timeout:", cfg.App.Timeout)
	fmt.Println("Concurrency:", cfg.App.Concurrency)
	fmt.Println("RateLimit RPS:", cfg.App.RateLimit.RPS)
	fmt.Println("RateLimit Burst:", cfg.App.RateLimit.Burst)
	fmt.Println("MetadataCache Dir:", cfg.App.MetadataCache.Dir)
	fmt.Println("MetadataCache TTL:", cfg.App.MetadataCache.TTL)

	// 你也可以读 output/network/watchlists
	fmt.Println("Console enabled:", cfg.Output.Console.Enabled)
	fmt.Println("Networks count:", len(cfg.Networks))
	fmt.Println("Watchlists count:", len(cfg.Watchlists))
}

func verify_config() {

}
