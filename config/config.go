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
	GlobalTimeout time.Duration `yaml:"global_timeout"`
	Retry         Retry         `yaml:"retry"`
	Timeout       time.Duration `yaml:"timeout"`
	Concurrency   int           `yaml:"concurrency"`
	BatchSize     int           `yaml:"batch_size"`
	RateLimit     RateLimit     `yaml:"rate_limit"`
	MetadataCache MetadataCache `yaml:"metadata_cache"`
}

type Retry struct {
	MaxRetries int           `yaml:"max_retries"`
	BaseDelay  time.Duration `yaml:"base_delay"`
	MaxDelay   time.Duration `yaml:"max_delay"`
}

type RateLimit struct {
	RPS   int `yaml:"rps"`
	Burst int `yaml:"burst"`
}

type MetadataCache struct {
	Dir      string        `yaml:"dir"`
	CachedAt int64         `json:"cached_at"` // 时间戳
	TTL      time.Duration `yaml:"ttl"`
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
	Enabled    bool   `yaml:"enabled"`
	Dir        string `yaml:"dir"`
	Mode       string `yaml:"mode"`
	FlushEvery int    `yaml:"flush_every"`
}
