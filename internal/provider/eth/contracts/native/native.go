package native

import (
	"context"
	"crypto-monitor/internal/provider"
	"crypto-monitor/tools"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Checker struct {
	EvmClient *ethclient.Client
}

func NewChecker(client *ethclient.Client) (*Checker, error) {
	return &Checker{
		EvmClient: client,
	}, nil
}

func (c *Checker) BalanceOf(ctx context.Context, timeout time.Duration, address common.Address) (*provider.TokenBalance, error) {
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// 做超时控制
	rawBalance, err := c.EvmClient.BalanceAt(ctx2, address, nil)
	if err != nil {
		return nil, fmt.Errorf("查询余额失败: %w", err)
	}
	balance := tools.FormatUnits(rawBalance, 18)

	return &provider.TokenBalance{
		Symbol:       "ETH",                                                             // TODO bsc链上原生代币是bnb
		TokenAddress: common.HexToAddress("0x0000000000000000000000000000000000000000"), // 表示0地址 写死不做校验
		Balance:      balance,
		RawBalance:   rawBalance,
		Owner:        address,
		Success:      true,
	}, err
}
