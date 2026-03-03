package engine

import (
	"context"
	"crypto-monitor/config"
	"crypto-monitor/internal/provider/eth"
	"crypto-monitor/internal/provider/eth/contracts/multicall3"
	"fmt"
	"os"
	"time"
)

func InitNetworks(ctx context.Context, cfg config.Root, timeout time.Duration) (map[string]*NetworkRuntime, error) {
	runtimes := make(map[string]*NetworkRuntime, len(cfg.Networks))

	for name, n := range cfg.Networks {
		if name == "" {
			return nil, fmt.Errorf("网络 %s: rpc配置为空", name)
		}
		var (
			client  *eth.EvmClient
			rpcUsed string
			err     error
		)
		for _, rpc := range n.RPC {
			rpc = os.ExpandEnv(rpc) // 支持 ${ETH_RPC_URL}
			// 连接EVM
			evmClient, err := eth.NewClient(rpc, ctx, timeout)
			if err == nil {
				client = evmClient
				rpcUsed = rpc
				break
			}
		}
		if client == nil {
			return nil, fmt.Errorf("网络 %s: 所有rpc连接都失败: %w", name, err)
		}
		// 防止rpc连接的网络不对应
		if n.ChainID != 0 && int(client.ChainID.Int64()) != n.ChainID {
			return nil, fmt.Errorf("网络 %s: chain_id 不匹配, cfg=%d node=%d", name, n.ChainID, client.ChainID.Int64())
		}
		multiChecker, err := multicall3.NewMultiChecker(client.Client)
		if err != nil {
			return nil, fmt.Errorf("网络 %s: 绑定 multicall 失败: %w", name, err)
		}
		runtimes[name] = &NetworkRuntime{
			Name:         name,
			ChainID:      int(client.ChainID.Int64()),
			NativeSymbol: n.NativeSymbol,
			RPCs:         n.RPC,
			RPCUsed:      rpcUsed,
			Client:       client,
			MultiChecker: multiChecker,
		}
	}
	return runtimes, nil
}
