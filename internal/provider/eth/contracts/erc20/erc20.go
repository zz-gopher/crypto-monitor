package erc20

import (
	"context"
	"crypto-monitor/internal/provider"
	"crypto-monitor/internal/provider/eth"
	"crypto-monitor/tools"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Checker struct {
	TokenAddress common.Address
	Token        *Erc20
}

// NewChecker 通过代币合约地址 绑定已经部署的合约
func NewChecker(tokenAddress common.Address, evmClient *eth.EvmClient) (*Checker, error) {
	token, err := NewErc20(tokenAddress, evmClient.Client)
	if err != nil {
		return nil, fmt.Errorf("绑定失败 %s: %w", tokenAddress.Hex(), err)
	}
	return &Checker{
		TokenAddress: tokenAddress,
		Token:        token,
	}, nil
}

// BalanceOf 查询代币余额信息，包括代币名称和代币精度
func (c *Checker) BalanceOf(ctx context.Context, timeout time.Duration, address common.Address) (*provider.TokenBalance, error) {
	opts, cancel := tools.CallOpts(ctx, timeout)
	defer cancel()
	rawBalance, err := c.Token.BalanceOf(opts, address)
	if err != nil {
		return nil, fmt.Errorf("查询余额失败: %w", err)
	}
	// TODO 后续存储到数据库或Redis 节省RPC的消耗
	symbol, err := tools.FetchSymbol(ctx, timeout, c.Token)
	if err != nil {
		return nil, err
	}
	decimals, err := c.GetDecimal(ctx, timeout)
	if err != nil {
		return nil, err
	}
	readableBalance := tools.FormatUnits(rawBalance, decimals)
	return &provider.TokenBalance{
		Symbol:       symbol,
		TokenAddress: c.TokenAddress,
		Balance:      readableBalance,
		RawBalance:   rawBalance,
		Decimals:     decimals,
		Owner:        address,
	}, nil
}

// GetDecimal 获取代币精度
func (c *Checker) GetDecimal(ctx context.Context, timeout time.Duration) (uint8, error) {
	opts, cancel := tools.CallOpts(ctx, timeout)
	defer cancel()
	// 带上超时控制
	decimals, err := c.Token.Decimals(opts)
	if err != nil {
		return 0, fmt.Errorf("获取代币精度失败 %s: %w", c.TokenAddress.Hex(), err)
	}
	return decimals, nil
}
