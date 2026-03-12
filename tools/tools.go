package tools

import (
	"context"
	"crypto-monitor/internal/provider"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// FormatUnits 把最小单位按 decimals 格式化成“人类单位”。
func FormatUnits(balance *big.Int, decimals uint8) *big.Float {
	// 1. 创建一个 big.Float 类型的余额副本
	fBalance := new(big.Float).SetInt(balance)

	// 2. 计算除数 10^decimals
	base := big.NewInt(10)
	power := big.NewInt(int64(decimals)) // 这里把 uint8 转为 int64
	divisorInt := new(big.Int).Exp(base, power, nil)

	// 3. 把除数也转为 big.Float
	fDivisor := new(big.Float).SetInt(divisorInt)

	// 4. 做除法 (Balance / Divisor)
	result := new(big.Float).Quo(fBalance, fDivisor)

	return result
}

// CallOpts 超时控制opts封装
func CallOpts(ctx context.Context, timeout time.Duration) (*bind.CallOpts, func()) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return &bind.CallOpts{Context: ctx}, cancel
}

// FetchSymbol ：通用查询 symbol，带超时控制。
// - caller: 任何实现了 SymbolCaller 的合约绑定对象（ERC20/ERC721 都可）
// - allowUnknown: 你可以选择失败时返回 "UNKNOWN" 还是把错误抛上去
func FetchSymbol(
	ctx context.Context,
	timeout time.Duration,
	caller provider.SymbolCaller,
) (string, error) {

	opts, cancel := CallOpts(ctx, timeout)
	defer cancel()

	symbol, err := caller.Symbol(opts)
	if err != nil {
		return "", fmt.Errorf("获取代币名称失败: %w", err)
	}
	return symbol, nil
}

// ShortAddress 地址缩略：0xFbE4...1234
func ShortAddress(addr common.Address) string {
	s := addr.Hex()
	if len(s) <= 10 {
		return s
	}
	return s[:6] + "..." + s[len(s)-4:]
}

// SplitAddresses 数组切片
func SplitAddresses(addresses []common.Address, batchSize int) [][]common.Address {
	if batchSize <= 0 {
		batchSize = len(addresses)
	}
	if len(addresses) == 0 {
		return nil
	}

	var batches [][]common.Address
	for i := 0; i < len(addresses); i += batchSize {
		end := i + batchSize
		if end > len(addresses) {
			end = len(addresses)
		}
		batches = append(batches, addresses[i:end])
	}
	return batches
}

// DecodeSymbol 解析 Symbol 的兼容函数
func DecodeSymbol(erc20Abi *abi.ABI, returnData []byte) (string, error) {
	// 检查数据是否为空（防空指针）
	if len(returnData) == 0 {
		return "UNKNOWN", nil
	}

	// 1. 尝试按标准 ERC20 (string) 解包
	var outString []any
	err := erc20Abi.UnpackIntoInterface(&outString, "symbol", returnData)
	if err == nil && len(outString) > 0 {
		return outString[0].(string), nil // 解包成功，直接返回！
	}

	// 2. 🚨 触发降级机制：如果 string 解包失败，尝试按 bytes32 解包！
	// 因为 go-ethereum 的 ABI 解包强依赖你在 abi.JSON 里定义的类型，
	// 我们不能直接用 ERC20 的 ABI 解了，必须自己硬解这 32 个字节！

	// 只要返回值长度大于等于 32，我们就强行把它当 bytes32 读出来
	if len(returnData) >= 32 {
		var bytes32Symbol [32]byte
		copy(bytes32Symbol[:], returnData[:32])

		// 把 [32]byte 转成字符串，并剔除末尾多余的空字符 (\x00)
		// 比如 "MKR\x00\x00\x00..." -> "MKR"
		cleanedSymbol := strings.TrimRight(string(bytes32Symbol[:]), "\x00")

		// 很多时候不仅是 null 字符，还可能有不可见的控制字符，安全起见可以用 TrimSpace
		cleanedSymbol = strings.TrimSpace(cleanedSymbol)

		if cleanedSymbol != "" {
			return cleanedSymbol, nil
		}
	}

	return "UNKNOWN", fmt.Errorf("无法解析 symbol, 既不是 string 也不是 bytes32")
}
