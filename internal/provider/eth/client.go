package eth

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	MaxRetries    = 3               // 最大重试次数
	RetryInterval = 1 * time.Second // 重试间隔
)

type EvmClient struct {
	RPC     string
	Client  *ethclient.Client
	ChainID *big.Int
}

// NewClient 创建并初始化一个 EvmClient，连接到指定的 RPC 节点
func NewClient(rpcUrl string, timeout time.Duration) (*EvmClient, error) {
	var lastErr error
	for i := 0; i < MaxRetries; i++ {
		var client *ethclient.Client
		var err error
		// 带超时的连接控制 (避免节点挂了导致程序一直卡在 Dial)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		client, err = ethclient.DialContext(ctx, rpcUrl)
		cancel()
		if err == nil {
			// 连接成功后，再查个 ChainID 确认节点真的活着
			cidCtx, cidCancel := context.WithTimeout(context.Background(), timeout)
			chainID, cidErr := client.ChainID(cidCtx)
			cidCancel()
			if cidErr == nil {
				// 彻底成功，返回对象
				return &EvmClient{
					Client:  client,
					ChainID: chainID,
					RPC:     rpcUrl,
				}, nil
			}
			client.Close()   // 关连接，避免泄露
			lastErr = cidErr // 如果 ChainID 失败，更新错误信息
		} else {
			lastErr = err
		}
		// 如果失败了，打印日志并等待
		fmt.Printf("⚠️ 连接失败 (尝试 %d/%d): %v. %s后重试...\n", i+1, MaxRetries, lastErr, RetryInterval)
		time.Sleep(RetryInterval)
	}
	// 都失败，彻底放弃
	return nil, fmt.Errorf("❌ 重试 %d 次后连接失败: %w", MaxRetries, lastErr)
}

// Close 关闭EVM连接
func (c *EvmClient) Close() {
	if c.Client != nil {
		c.Client.Close()
	}
}

// IsConnected EVM是否链接成功
func (c *EvmClient) IsConnected() bool {
	_, err := c.Client.BlockNumber(context.Background())
	return err == nil
}
