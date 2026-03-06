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
	// 展开 ${ETH_RPC_URL} / ${ARB_RPC_URL} 这类占位符
	expanded := os.ExpandEnv(string(data))
	if hasUnexpandedEnv(expanded) {
		return nil, fmt.Errorf("config contains unexpanded env vars, please set them in .env or system env")
	}
	var cfg Root
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml unmarshal %s: %w", abs, err)
	}

	return &cfg, nil
}

// LoadAddressesFromTXT 读取文件并返回去重后的地址列表
func LoadAddressesFromTXT(pathOrGlob string) ([]common.Address, error) {
	files, err := resolveFiles(pathOrGlob)
	if err != nil {
		return nil, err
	}

	// 去重 map
	seen := make(map[common.Address]struct{})
	var addresses []common.Address

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
func loadOneTXTFile(filePath string, seen map[common.Address]struct{}) ([]common.Address, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var items []common.Address

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
		items = append(items, addr)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", filePath, err)
	}

	return items, nil
}

func hasUnexpandedEnv(s string) bool {
	// 很粗暴但好用：只要还有 ${ 就认为没替换干净
	return contains(s, "${")
}
func contains(s, sub string) bool {
	return len(sub) > 0 && (len(s) >= len(sub)) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
