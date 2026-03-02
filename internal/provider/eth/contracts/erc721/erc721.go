package erc721

import (
	"context"
	"crypto-monitor/internal/provider"
	"crypto-monitor/internal/provider/eth"
	"crypto-monitor/tools"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Checker struct {
	TokenAddress common.Address
	Token        *Erc721
}

// NewChecker 通过代币合约地址绑定已经部署的合约
func NewChecker(tokenAddress common.Address, evmClient *eth.EvmClient) (*Checker, error) {
	token, err := NewErc721(tokenAddress, evmClient.Client)
	if err != nil {
		return nil, fmt.Errorf("绑定失败 %s: %w", tokenAddress.Hex(), err)
	}
	return &Checker{
		TokenAddress: tokenAddress,
		Token:        token,
	}, nil
}

// BalanceOf  查询指定地址ERC721代币,没有精度不需要转换
func (c *Checker) BalanceOf(ctx context.Context, timeout time.Duration, address common.Address) (*provider.TokenBalance, error) {
	opts, cancel := tools.CallOpts(ctx, timeout)
	defer cancel()
	rawBalance, err := c.Token.BalanceOf(opts, address)
	if err != nil {
		return nil, fmt.Errorf("查询余额失败: %w", err)
	}
	symbol, err := c.Token.Symbol(nil)
	if err != nil {
		symbol = "UNKNOWN"
		return nil, err
	}
	return &provider.TokenBalance{
		Symbol:       symbol,
		TokenAddress: c.TokenAddress,
		Balance:      new(big.Float).SetInt(rawBalance),
		RawBalance:   rawBalance,
		Decimals:     1,
		Owner:        address,
	}, nil
}
