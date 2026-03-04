package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

// AddressItem 代表一个地址
type AddressItem struct {
	Address common.Address
}

// LoadCfg Load 从指定路径读取 config.yaml 并解析为 Root
func LoadCfg(path string) (*Root, error) {
	if path == "" {
		return nil, errors.New("config path is empty")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get abs path: %w", err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", abs, err)
	}

	var cfg Root
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml unmarshal %s: %w", abs, err)
	}

	return &cfg, nil
}

// LoadAddressesFromTXT 读取文件并返回去重后的地址列表
func LoadAddressesFromTXT(pathOrGlob string) ([]AddressItem, error) {
	files, err := resolveFiles(pathOrGlob)
	if err != nil {
		return nil, err
	}

	// 去重 map
	seen := make(map[common.Address]struct{})
	var addresses []AddressItem

	// 遍历文件，读取每个地址
	for _, file := range files {
		items, err := loadOneTXTFile(file, seen)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, items...)
	}

	return addresses, nil
}

// resolveFiles 解析 glob 模式（比如 `*.txt`）
func resolveFiles(path string) ([]string, error) {
	// 如果包含通配符就执行 glob
	if strings.ContainsAny(path, "*?[]") {
		files, err := filepath.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", path, err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("地址 %s: 没有找到任何含.txt文件", path)
		}
		sort.Strings(files) // 稳定顺序
		return files, nil
	}
	// 否则当作单文件
	return []string{path}, nil
}

// loadOneTXTFile 逐行读取每个 txt 文件
func loadOneTXTFile(filePath string, seen map[common.Address]struct{}) ([]AddressItem, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var items []AddressItem

	// 逐行读取文件
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // 跳过空行和注释行
		}

		// 校验地址格式
		if !common.IsHexAddress(line) {
			continue // 非法地址跳过
		}
		addr := common.HexToAddress(line)

		// 如果已存在，跳过
		if _, exists := seen[addr]; exists {
			continue
		}
		seen[addr] = struct{}{} // 添加到 seen

		// 加入地址列表
		items = append(items, AddressItem{Address: addr})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", filePath, err)
	}

	return items, nil
}
