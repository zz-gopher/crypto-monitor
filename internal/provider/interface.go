package provider

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// TokenBalance 统一的返回结果结构
type TokenBalance struct {
	Symbol       string         // 代币符号
	TokenAddress common.Address // 代币合约地址
	Balance      *big.Float     // 人类可读余额
	RawBalance   *big.Int       // 原始余额
	Owner        common.Address // 代币所有者
}

// AssetChecker 定义通用的查余额接口
type AssetChecker interface {
	BalanceOf(address common.Address) (*TokenBalance, error)
}
