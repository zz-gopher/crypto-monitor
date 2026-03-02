package erc20

import (
	"crypto-monitor/internal/provider"
	"crypto-monitor/internal/provider/eth"
	"crypto-monitor/tools"
	"fmt"

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

func (c *Checker) BalanceOf(address common.Address) (*provider.TokenBalance, error) {
	rawBalance, err := c.Token.BalanceOf(nil, address)
	if err != nil {
		return nil, fmt.Errorf("查询余额失败: %w", err)
	}
	// TODO 后续存储到数据库 节省RPC的消耗
	symbol, err := c.Token.Symbol(nil)
	if err != nil {
		symbol = "UNKNOWN"
	}
	decimals, err := c.Token.Decimals(nil)
	if err != nil {
		return nil, fmt.Errorf("获取代币精度失败 %s: %w", c.TokenAddress.Hex(), err)
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
