package multicall3

import (
	"context"
	"crypto-monitor/internal/provider"
	"crypto-monitor/internal/provider/eth/contracts/erc20"
	"crypto-monitor/internal/provider/eth/contracts/erc721"
	"crypto-monitor/tools"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	AssetTypeNative = "native"
	AssetTypeERC20  = "erc20"
	AssetTypeERC721 = "erc721"
)

// ContractAddress multicall3合约地址
const ContractAddress = "0xcA11bde05977b3631167028862bE2a173976CA11"

type MultiChecker struct {
	Client    *ethclient.Client
	Multicall *Multicall3
	MultiAddr common.Address
}
type callItem struct {
	TokenAddr common.Address
	Owner     common.Address
	Type      string
	CallData  []byte
	AbiName   string
}

// NewMultiChecker 绑定multicall3合约
func NewMultiChecker(client *ethclient.Client) (*MultiChecker, error) {
	multicallAddr := common.HexToAddress(ContractAddress)

	// 绑定multicall合约
	multi, err := NewMulticall3(multicallAddr, client)
	if err != nil {
		return nil, err
	}
	return &MultiChecker{
		Client:    client,
		Multicall: multi,
		MultiAddr: multicallAddr,
	}, nil
}

// CheckToken multicall3是一个批量打包多个 eth_call，用一次 RPC 拿回全部结果，能够显著加速并降低限流压力。
func (m *MultiChecker) CheckToken(
	tType string,
	tokenAddr common.Address,
	ctx context.Context,
	timeout time.Duration,
	addresses []common.Address,
) ([]provider.TokenBalance, error) {

	var decimals uint8
	var symbol string
	var callList []callItem
	var balances []provider.TokenBalance
	multicallAddr := common.HexToAddress(ContractAddress)
	// 准备 Multicall3 的 ABI，用于 Native 代币打包
	mcAbi, err := Multicall3MetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get multicall abi: %w", err)
	}
	switch tType {
	case AssetTypeNative:
		symbol = "ETH"
		decimals = 18
		for _, owner := range addresses {
			callData, err := mcAbi.Pack("getEthBalance", owner)
			if err != nil {
				return nil, fmt.Errorf("pack getEthBalance failed: %w", err)
			}

			callList = append(callList, callItem{
				TokenAddr: tokenAddr, // 这里的 TokenAddr 只是标识，Target 在下面会换成 MulticallAddr
				Owner:     owner,
				Type:      tType,
				CallData:  callData,
				AbiName:   "getEthBalance",
			})
		}
	case AssetTypeERC721:
		decimals = 0
		erc721Checker, err := erc721.NewChecker(tokenAddr, m.Client)
		if err != nil {
			return nil, err
		}
		symbol, err = tools.FetchSymbol(ctx, timeout, erc721Checker.Token)
		if err != nil {
			symbol = "UNKNOWN"
		}
		items, err := setCallList(erc721.Erc721MetaData, addresses, tokenAddr, tType)
		if err != nil {
			return nil, err
		}
		callList = append(callList, items...)
	case AssetTypeERC20:
		items, err := setCallList(erc20.Erc20MetaData, addresses, tokenAddr, tType)
		if err != nil {
			return nil, err
		}
		callList = append(callList, items...)
		// 绑定erc20合约
		erc20Checker, err := erc20.NewChecker(tokenAddr, m.Client)
		if err != nil {
			return nil, fmt.Errorf("failed to bind token %s: %w", tokenAddr.Hex(), err)
		}
		symbol, err = tools.FetchSymbol(ctx, timeout, erc20Checker.Token)
		if err != nil {
			return nil, err
		}
		// 查询代币精度
		decimals, err = erc20Checker.GetDecimal(ctx, timeout)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("未知代币类型")
	}
	var mcCalls []Multicall3Call3
	for _, c := range callList {
		target := c.TokenAddr
		// 关键修正：如果查原生代币，Target 必须是 Multicall3 合约地址本身
		if c.Type == AssetTypeNative {
			target = multicallAddr
		}

		mcCalls = append(mcCalls, Multicall3Call3{
			Target:       target,
			CallData:     c.CallData,
			AllowFailure: true, // 允许部分失败
		})
	}
	// 执行multicall3的Aggregate3,把多个合约调用封装（Pack）成一个大调用，一次性发给区块链执行
	opts, cancel := tools.CallOpts(ctx, timeout)
	defer cancel()
	// rpc超时控制
	resp, err := m.Multicall.Aggregate3(opts, mcCalls)
	if err != nil {
		return nil, fmt.Errorf("multicall aggregate3 failed: %w", err)
	}
	erc20Abi, _ := erc20.Erc20MetaData.GetAbi()
	erc721Abi, _ := erc721.Erc721MetaData.GetAbi()

	for i, res := range resp {
		if len(resp) != len(callList) {
			return nil, fmt.Errorf("aggregate3 返回数量不一致")
		}
		req := callList[i]
		tb := provider.TokenBalance{
			TokenAddress: req.TokenAddr,
			Owner:        req.Owner,
			Balance:      big.NewFloat(0), // 默认为 0
			Symbol:       symbol,
		}
		// 检查是否调用成功
		if !res.Success {
			balances = append(balances, tb)
			continue
		}
		// 检查返回数据是否为空
		if len(res.ReturnData) == 0 {
			// ERC20可能虽然没查到数据，但我们可以“认为”它余额是 0
			// 因为一个不存在的合约，当然没有它的币
			balances = append(balances, tb)
			continue
		}
		// 视为查询成功
		tb.Success = true
		// 根据类型解码
		var decodeErr error
		var rawBalance *big.Int
		switch req.Type {
		case AssetTypeERC20:
			// ERC20 解码
			rawBalance, decodeErr = decodeUint256(erc20Abi, "balanceOf", res.ReturnData)
			tb.Balance = tools.FormatUnits(rawBalance, decimals)
			tb.RawBalance = rawBalance
			tb.Decimals = decimals
		case AssetTypeERC721:
			// ERC721 解码
			rawBalance, decodeErr = decodeUint256(erc721Abi, "balanceOf", res.ReturnData)
			tb.Balance = new(big.Float).SetInt(rawBalance)
			tb.RawBalance = rawBalance
			tb.Decimals = decimals
		case AssetTypeNative:
			// Native 解码 (getEthBalance 返回 uint256)
			// 直接由 bytes 转 bigInt 即可，或者用 ABI unpack 也可以
			var out []any
			out, decodeErr = mcAbi.Unpack("getEthBalance", res.ReturnData)
			if decodeErr != nil {
				return nil, decodeErr
			}
			rawBalance = out[0].(*big.Int)
			tb.Balance = tools.FormatUnits(rawBalance, 18)
			tb.RawBalance = rawBalance
			tb.Decimals = decimals
		}
		if decodeErr != nil {
			return nil, decodeErr
		}

		balances = append(balances, tb)
	}
	return balances, nil
}

// 辅助函数：修改 setCallList 以接受 type
func setCallList(metaData *bind.MetaData, owners []common.Address, tokenAddr common.Address, tType string) ([]callItem, error) {
	var callList []callItem
	parsed, err := metaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("parse abi: %w", err)
	}
	for _, owner := range owners {
		// 把函数+参数->编码成EVM需要的calldata
		data, err := parsed.Pack("balanceOf", owner)
		if err != nil {
			return nil, fmt.Errorf("pack balanceOf: %w", err)
		}
		callList = append(callList, callItem{
			TokenAddr: tokenAddr,
			Owner:     owner,
			Type:      tType, // 使用传入的 type
			CallData:  data,
			AbiName:   "balanceOf",
		})
	}
	return callList, nil
}

// 辅助函数：通用解码 Uint256
func decodeUint256(parsedAbi *abi.ABI, method string, data []byte) (*big.Int, error) {
	// 1. 解码二进制数据
	unpacked, err := parsedAbi.Unpack(method, data)
	if err != nil {
		return nil, err
	}

	// 2. 校验数据完整性
	if len(unpacked) == 0 {
		return nil, errors.New("no data unpacked")
	}

	// 3. 类型断言 (Type Assertion)
	balance, ok := unpacked[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("result is not *big.Int, type is %T", unpacked[0])
	}
	return balance, nil
}
