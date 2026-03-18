package config

import "time"

// Root 对应整个配置文件最外层（匹配你最新 YAML：networks + tokens + watchlists）
type Root struct {
	App        AppConfig              `yaml:"app" validate:"required"`
	Output     OutputConfig           `yaml:"output" validate:"required"`
	Networks   map[string]NetworkItem `yaml:"networks" validate:"required,min=1,dive"`   // 至少配1个网络 key: ethereum-mainnet / arbitrum-one / ...
	Tokens     map[string]TokenDef    `yaml:"tokens" validate:"dive"`                    // key: USDT / USDC / ...
	Watchlists []Watchlist            `yaml:"watchlists" validate:"required,min=1,dive"` // dive表示深入校验切片内部
}

// AppConfig app: {...}
type AppConfig struct {
	Retry         Retry         `yaml:"retry" validate:"required"`
	GlobalTimeout time.Duration `yaml:"global_timeout" validate:"required,gt=0"`
	Timeout       time.Duration `yaml:"timeout" validate:"required,gt=0"`
	Concurrency   int           `yaml:"concurrency" validate:"required,gt=0,lte=1000"` // 最大并发限制在 1000 以内防搞崩机器
	BatchSize     int           `yaml:"batch_size" validate:"required,gt=0"`
	RateLimit     RateLimit     `yaml:"rate_limit" validate:"required"`
	MetadataCache MetadataCache `yaml:"metadata_cache" validate:"required"`
}

type Retry struct {
	MaxRetries int           `yaml:"max_retries" validate:"gte=0"`
	BaseDelay  time.Duration `yaml:"base_delay" validate:"required,gt=0"`
	MaxDelay   time.Duration `yaml:"max_delay" validate:"required,gtefield=BaseDelay"` // max_delay 必须大于等于 base_delay
}

type RateLimit struct {
	RPS   int `yaml:"rps" validate:"required,gt=0"`
	Burst int `yaml:"burst" validate:"required,gte=1"` // 突发至少得是1吧
}

type MetadataCache struct {
	Dir      string        `yaml:"dir" validate:"required"`
	CachedAt int64         `json:"cached_at"` // 时间戳
	TTL      time.Duration `yaml:"ttl" validate:"required,gt=0"`
}

// NetworkItem networks: { <name>: {...}, ... }
type NetworkItem struct {
	ChainID      int      `yaml:"chain_id" validate:"required,gt=0"`
	RPC          []string `yaml:"rpc" validate:"required,min=1"`
	NativeSymbol string   `yaml:"native_symbol" validate:"required"`
}

// TokenDef tokens: { USDT: {...}, ... }
type TokenDef struct {
	Type       string                    `yaml:"type" validate:"required,oneof=erc20 erc721"` // 限制类型
	PerNetwork map[string]TokenOnNetwork `yaml:"per_network" validate:"required,min=1,dive"`
}

type TokenOnNetwork struct {
	Contract string `yaml:"contract" validate:"required,startswith=0x,len=42"` // 必须是 0x 开头且长度为 42 的 EVM 规范地址
}

// Watchlist watchlists: [...]
type Watchlist struct {
	Name        string     `yaml:"name" validate:"required"`
	Networks    []string   `yaml:"networks" validate:"required,min=1,dive"`
	AddressGlob string     `yaml:"address_glob" validate:"required"`
	Assets      []AssetRef `yaml:"assets" validate:"required,min=1,dive"`
}

// AssetRef watchlists.assets: 支持 native / token
type AssetRef struct {
	Token string `yaml:"token" validate:"required"`
}

type OutputConfig struct {
	CSV CSV `yaml:"csv" validate:"required"`
}

type CSV struct {
	Enabled    bool   `yaml:"enabled"`
	Dir        string `yaml:"dir" validate:"required_if=Enabled true"`                         // 只有开启CSV时，目录才是必填的！
	Mode       string `yaml:"mode" validate:"required_if=Enabled true,oneof=append overwrite"` // 只能是这俩词
	FlushEvery int    `yaml:"flush_every" validate:"required_if=Enabled true,gt=0"`            // 刷盘频率必须大于 0
}
