package native

import (
	"context"
	"crypto-monitor/internal/provider"
	"crypto-monitor/internal/provider/eth"
	"crypto-monitor/tools"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Checker struct {
	EvmClient *ethclient.Client
}

func NewChecker(evmClient *eth.EvmClient) (*Checker, error) {
	return &Checker{
		EvmClient: evmClient.Client,
	}, nil
}

func (c *Checker) BalanceOf(address common.Address) (*provider.TokenBalance, error) {
	rawBalance, err := c.EvmClient.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return nil, fmt.Errorf("查询余额失败: %w", err)
	}
	balance := tools.FormatUnits(rawBalance, 18)

	return &provider.TokenBalance{
		Symbol:       "ETH",
		TokenAddress: common.HexToAddress("0x0000000000000000000000000000000000000000"), // 表示0地址 写死不做校验
		Balance:      balance,
		RawBalance:   rawBalance,
		Owner:        address,
	}, err
}
