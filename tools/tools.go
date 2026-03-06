package tools

import (
	"context"
	"crypto-monitor/internal/provider"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// FormatUnits 把最小单位按 decimals 格式化成“人类单位”。
func FormatUnits(balance *big.Int, decimals uint8) *big.Float {
	// 1. 创建一个 big.Float 类型的余额副本
	fBalance := new(big.Float).SetInt(balance)

	// 2. 计算除数 10^decimals
	base := big.NewInt(10)
	power := big.NewInt(int64(decimals)) // 这里把 uint8 转为 int64
	divisorInt := new(big.Int).Exp(base, power, nil)

	// 3. 把除数也转为 big.Float
	fDivisor := new(big.Float).SetInt(divisorInt)

	// 4. 做除法 (Balance / Divisor)
	result := new(big.Float).Quo(fBalance, fDivisor)

	return result
}

// CallOpts 超时控制opts封装
func CallOpts(ctx context.Context, timeout time.Duration) (*bind.CallOpts, func()) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return &bind.CallOpts{Context: ctx}, cancel
}

// FetchSymbol ：通用查询 symbol，带超时控制。
// - caller: 任何实现了 SymbolCaller 的合约绑定对象（ERC20/ERC721 都可）
// - allowUnknown: 你可以选择失败时返回 "UNKNOWN" 还是把错误抛上去
func FetchSymbol(
	ctx context.Context,
	timeout time.Duration,
	caller provider.SymbolCaller,
) (string, error) {

	opts, cancel := CallOpts(ctx, timeout)
	defer cancel()

	symbol, err := caller.Symbol(opts)
	if err != nil {
		return "", fmt.Errorf("获取代币名称失败: %w", err)
	}
	return symbol, nil
}

// ShortAddress 地址缩略：0xFbE4...1234
func ShortAddress(addr common.Address) string {
	s := addr.Hex()
	if len(s) <= 10 {
		return s
	}
	return s[:6] + "..." + s[len(s)-4:]
}
