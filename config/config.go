package config

import "time"

// Root 对应整个配置文件最外层（匹配你最新 YAML：networks + tokens + watchlists）
type Root struct {
	App        AppConfig              `yaml:"app"`
	Output     OutputConfig           `yaml:"output"`
	Networks   map[string]NetworkItem `yaml:"networks"` // key: ethereum-mainnet / arbitrum-one / ...
	Tokens     map[string]TokenDef    `yaml:"tokens"`   // key: USDT / USDC / ...
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

// TokenDef tokens: { USDT: {...}, ... }
type TokenDef struct {
	Type       string                    `yaml:"type"`        // erc20 / erc721 / ...
	PerNetwork map[string]TokenOnNetwork `yaml:"per_network"` // key: network name
}

type TokenOnNetwork struct {
	Contract string `yaml:"contract"`
}

// Watchlist watchlists: [...]
type Watchlist struct {
	Name        string     `yaml:"name"`
	Networks    []string   `yaml:"networks"`
	AddressGlob string     `yaml:"address_glob"`
	Assets      []AssetRef `yaml:"assets"` // 要查询哪些资产
}

// AssetRef watchlists.assets: 支持 native / token
type AssetRef struct {
	// 1) 原生资产：写 type: native
	Type string `yaml:"type,omitempty"` // native

	// 2) 引用 token：写 token: USDT
	Token string `yaml:"token,omitempty"`
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
