package retry

import (
	"context"
	"crypto-monitor/config"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

// Do 指数级退避重试算法
func Do(ctx context.Context, cfg config.Retry, operation func() error) error {
	maxRetries := cfg.MaxRetries // 最大重试次数
	baseDelay := cfg.BaseDelay   // 基础重试时间
	maxDelay := cfg.MaxDelay     // 最大重试时间
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 执行真正的业务逻辑
		err := operation()
		if err == nil {
			return nil // 成功直接返回
		}
		if attempt == maxRetries {
			return fmt.Errorf("达到最大重试次数 %d: %w", maxRetries, err)
		}
		// 判断是否是可重式异常
		if !isRetryableError(err) {
			return err
		}
		// 计算指数退避时间: base * 2^attempt
		delay := float64(baseDelay) * math.Pow(2, float64(attempt))
		if time.Duration(delay) > maxDelay {
			delay = float64(maxDelay)
		}
		// 加入 Jitter (比如加减 20% 的随机时间)，防止多个协程同时启动 将节点瞬间再次打挂
		jitter := (rand.Float64()*0.4 - 0.2) * delay
		finalDelay := time.Duration(delay + jitter)
		fmt.Printf("⚠️ 遇到错误，准备在 %v 后进行第 %d 次重试. 错误: %v\n", finalDelay, attempt+1, err)
		// 开启定时器
		timer := time.NewTimer(finalDelay)

		select {
		case <-timer.C: // 监听定时器倒计时
			// 睡醒了 进行下一次循环重试
		case <-ctx.Done():
			// 睡觉途中接到上级取消命令，计时器从内存删除，防止占用过多的内存
			timer.Stop()
			return fmt.Errorf("重试被上下文取消: %w", ctx.Err())
		}
	}
	return nil
}

// 判断是否可重试异常
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// --------------------------------
	// Go 原生标准错误类型判断 (使用 errors.Is / errors.As)
	// --------------------------------

	// 上下文被主动取消, 不重试
	if errors.Is(err, context.Canceled) {
		return false
	}

	// 上下文超时, 可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// 底层网络错误 (比如 TCP 连接断开、DNS 解析失败等), 可重试
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// --------------------------------------------
	// Web3 RPC专属的“脏数据”字符串匹配
	// --------------------------------------------
	errStr := strings.ToLower(err.Error())

	// 不重试的致命错误,后续有其他类型也可添加
	fatalKeywords := []string{
		"execution reverted", // 合约执行失败/回滚 (代码逻辑问题)
		"invalid argument",   // 参数错误
		"missing trie node",  // 节点历史数据丢失 (通常发生在查很老的区块时)
		"method not found",   // 节点不支持这个 RPC 方法
		"invalid address",    // 地址格式错
	}
	for _, keyword := range fatalKeywords {
		if strings.Contains(errStr, keyword) {
			return false
		}
	}
	// 可重试的错误，后续有其他类型也可添加
	retryKeywords := []string{
		"too many requests", // HTTP 429 限流 (最常见)
		"rate limit",        // 也是限流
		"timeout",           // 各种千奇百怪的超时
		"connection reset",  // 连接被强行重置
		"eof",               // 意外的流结束
		"502", "503", "504", // 各种网关和服务器负载错误
		"server error",     // 节点内部偶发错误
		"header not found", // 区块头暂时没同步过来
	}
	for _, keyword := range retryKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	// 未知的错误，默认不重试（安全第一，防止死循环）
	return false
}
