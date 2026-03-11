package metadata

type TokenMetadata struct {
	ChainID  string `json:"chain_id"`  // 链id
	Address  string `json:"address"`   // 代币地址
	Symbol   string `json:"symbol"`    // 代币symbol
	Name     string `json:"name"`      // 代币名称
	Decimals uint8  `json:"decimals"`  // 代币精度
	CachedAt int64  `json:"cached_at"` // 生产日期 (Unix秒)
}
