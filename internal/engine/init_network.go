package engine

import (
	"context"
	"crypto-monitor/config"
	"crypto-monitor/internal/provider/eth"
	"crypto-monitor/internal/provider/eth/contracts/multicall3"
	"errors"
	"fmt"
	"os"
	"time"
)

// InitNetworks 初始化配置 连接rpc网络
func InitNetworks(ctx context.Context, cfg *config.Root, timeout time.Duration) (map[string]*NetworkRuntime, map[string]error, error) {
	runtimes := make(map[string]*NetworkRuntime, len(cfg.Networks))
	failed := make(map[string]error)
	for name, n := range cfg.Networks {
		if len(n.RPC) == 0 {
			failed[name] = fmt.Errorf("rpc 列表为空")
			continue
		}
		var (
			client  *eth.EvmClient
			rpcUsed string
			lastErr error
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
			lastErr = err
		}
		if client == nil {
			failed[name] = fmt.Errorf("网络 %s: 所有rpc连接都失败: %w", name, lastErr)
			continue
		}
		// 防止rpc连接的网络不对应
		if n.ChainID != 0 && int(client.ChainID.Int64()) != n.ChainID {
			client.Close()
			failed[name] = fmt.Errorf("网络 %s: chain_id 不匹配, cfg=%d node=%d", name, n.ChainID, client.ChainID.Int64())
			continue
		}
		multiChecker, err := multicall3.NewMultiChecker(client.Client)
		if err != nil {
			client.Close()
			failed[name] = fmt.Errorf("网络 %s: 绑定 multicall 失败: %w", name, err)
			continue
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
	if len(runtimes) == 0 {
		return nil, failed, errors.New("无可用网络：全部初始化失败")
	}
	return runtimes, failed, nil
}
