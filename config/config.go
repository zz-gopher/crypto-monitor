package config

import "time"

// Root 对应整个配置文件最外层
type Root struct {
	App        AppConfig              `yaml:"app"`
	Output     OutputConfig           `yaml:"output"`
	Networks   map[string]NetworkItem `yaml:"networks"` // key 是 ethereum-mainnet / arbitrum-one
	Watchlists []Watchlist            `yaml:"watchlists"`
}

// AppConfig app: {...}
type AppConfig struct {
	PollInterval  time.Duration `yaml:"poll_interval"`
	Timeout       time.Duration `yaml:"timeout"`
	Concurrency   int           `yaml:"concurrency"`
	RateLimit     RateLimit     `yaml:"rate_limit"`
	MetadataCache MetadataCache `yaml:"metadata_cache"`
}

type RateLimit struct {
	RPS   int `yaml:"rps"`
	Burst int `yaml:"burst"`
}

type MetadataCache struct {
	Dir string        `yaml:"dir"`
	TTL time.Duration `yaml:"ttl"`
}

// NetworkItem networks: { <name>: {...}, ... }
type NetworkItem struct {
	ChainID      int      `yaml:"chain_id"`
	RPC          []string `yaml:"rpc"`
	NativeSymbol string   `yaml:"native_symbol"`
}

// Watchlist watchlists: [...]
type Watchlist struct {
	Name          string   `yaml:"name"`
	Networks      []string `yaml:"networks"`
	AddressSource string   `yaml:"address_source"`
	Assets        []Asset  `yaml:"assets"`
}

// Asset assets: [...]
type Asset struct {
	Type     string   `yaml:"type"`               // native / erc20
	Contract string   `yaml:"contract,omitempty"` // native 时为空
	Networks []string `yaml:"networks,omitempty"` // 可选：限制在哪些网络查询
}

type OutputConfig struct {
	Console Console `yaml:"console"`
	CSV     CSV     `yaml:"csv"`
}

type Console struct {
	Enabled  bool   `yaml:"enabled"`
	Format   string `yaml:"format"`
	Decimals int    `yaml:"decimals"`
	ShowWei  bool   `yaml:"show_wei"`
}

type CSV struct {
	Enabled    bool     `yaml:"enabled"`
	Path       string   `yaml:"path"`
	Mode       string   `yaml:"mode"`
	FlushEvery int      `yaml:"flush_every"`
	Columns    []string `yaml:"columns"`
}
