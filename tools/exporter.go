package tools

import (
	"crypto-monitor/config"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// 固定表头
var exportHeaders = []string{"链(Chain)", "代币地址(Token)", "钱包地址(Owner)", "代币名称(Symbol)", "余额(Balance)", "精度(Decimals)", "是否成功"}

// CSVExporter 生产级流式导出器
type CSVExporter struct {
	file       *os.File
	writer     *csv.Writer
	count      int
	flushEvery int
	mu         sync.Mutex // 加锁，防止多个并发协程同时写把文件写乱
}

// NewCSVExporter 初始化导出器
func NewCSVExporter(cfg config.CSV, path string) (*CSVExporter, error) {
	if !cfg.Enabled {
		return nil, nil // 如果没开启，直接返回 nil
	}

	// 确保上级目录存在 (比如 ./output 文件夹)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 根据 Mode 决定打开文件的姿势
	flag := os.O_CREATE | os.O_WRONLY
	if cfg.Mode == "append" {
		flag |= os.O_APPEND // 追加模式
	} else {
		flag |= os.O_TRUNC // 覆盖模式 (清空重写)
	}

	// 打开文件
	file, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开 CSV 文件失败: %w", err)
	}

	// 4. 判断是否需要写 BOM 头和表头
	fileInfo, _ := file.Stat()
	writer := csv.NewWriter(file)

	// 如果是新文件，或者是覆盖模式，我们需要写表头
	if fileInfo.Size() == 0 || cfg.Mode == "overwrite" {
		_, _ = file.WriteString("\xEF\xBB\xBF") // UTF-8 BOM，防止 Excel 乱码
		if err := writer.Write(exportHeaders); err != nil {
			return nil, fmt.Errorf("写入表头失败: %w", err)
		}
		writer.Flush()
	}

	return &CSVExporter{
		file:       file,
		writer:     writer,
		flushEvery: cfg.FlushEvery,
		count:      0,
	}, nil
}

// WriteRow 并发安全地写入单行数据
func (c *CSVExporter) WriteRow(row []string) error {
	if c == nil {
		return nil // 未开启 CSV 导出时，直接忽略
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 写入内存缓冲
	if err := c.writer.Write(row); err != nil {
		return err
	}
	c.count++

	// 达到阈值，执行刷盘
	if c.count >= c.flushEvery {
		c.writer.Flush()
		if err := c.writer.Error(); err != nil {
			return err
		}
		c.count = 0 // 重新计数
	}
	return nil
}

// Close 程序退出前调用，确保最后不满 flush_every 的数据也能存下来
func (c *CSVExporter) Close() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.writer.Flush() // 把最后一点残留刷入硬盘
	c.file.Close()   // 关掉文件
}
