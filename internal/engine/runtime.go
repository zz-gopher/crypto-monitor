package engine

import (
	"crypto-monitor/internal/provider/eth"
	"crypto-monitor/internal/provider/eth/contracts/multicall3"
)

type NetworkRuntime struct {
	Name         string
	ChainID      int    // 链上id
	NativeSymbol string // 链原生代币名称

	// 连接信息
	RPCs    []string // 链RPC集合，方便后续轮询
	RPCUsed string

	// 客户端
	Client       *eth.EvmClient           // 封装的client（含 ChainID / RPC / Client
	MultiChecker *multicall3.MultiChecker // abigen 绑定的 multicall 合约实例
}
