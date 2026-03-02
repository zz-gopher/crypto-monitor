package provider

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// TokenBalance 统一的返回结果结构
type TokenBalance struct {
	Symbol       string         // 代币符号
	TokenAddress common.Address // 代币合约地址
	Balance      *big.Float     // 人类可读余额
	Decimals     uint8          // 代币精度
	RawBalance   *big.Int       // 原始余额
	Owner        common.Address // 代币所有者
	Network      string         // 网络
	Success      bool           // 是否查询成功
}

// AssetChecker 定义通用的查余额接口
type AssetChecker interface {
	BalanceOf(ctx context.Context, timeout time.Duration, address common.Address) (*TokenBalance, error)
}

// SymbolCaller 定义通用的查Symbol接口
type SymbolCaller interface {
	Symbol(opts *bind.CallOpts) (string, error)
}
