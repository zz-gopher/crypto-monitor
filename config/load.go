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
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var validate *validator.Validate

// LoadCfg Load 从指定路径读取 config.yaml 并解析为 Root
func LoadCfg(path string) (*Root, error) {
	if path == "" {
		return nil, errors.New("配置路径为空")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败 %s: %w", abs, err)
	}
	// 展开 ${ETH_RPC_URL} / ${ARB_RPC_URL} 这类占位符
	expanded := os.ExpandEnv(string(data))
	if hasUnexpandedEnv(expanded) {
		return nil, fmt.Errorf("配置文件包含未展开的环境变量，请在.env或系统环境中设置它们")
	}
	var cfg Root
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败 %s: %w", abs, err)
	}
	// 初始化验证器并执行校验
	validate = validator.New()
	if err := validate.Struct(&cfg); err != nil {
		// 如果校验失败，格式化输出易读的错误信息
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return nil, err
		}

		fmt.Println("❌ 配置文件存在错误，请检查 config.yaml:")
		for _, err := range err.(validator.ValidationErrors) {
			// 获取出错的字段名
			fieldName := err.StructNamespace()

			// 根据不同的错误标签，翻译成中文提示
			var detail string
			switch err.Tag() {
			case "required":
				detail = "此项为必填，不能为空"
			case "min":
				// 如果是切片，说明数量不够
				detail = fmt.Sprintf("长度或数量不足 (要求至少 %s)", err.Param())
			case "gt":
				detail = fmt.Sprintf("必须大于 %s", err.Param())
			case "oneof":
				detail = fmt.Sprintf("输入非法，必须是 [%s] 之一", err.Param())
			case "startswith":
				detail = fmt.Sprintf("格式错误，必须以 %s 开头", err.Param())
			case "len":
				detail = fmt.Sprintf("长度不固定，必须等于 %s 位", err.Param())
			default:
				detail = fmt.Sprintf("校验失败 (规则: %s)", err.Tag())
			}

			fmt.Printf(" 👉 [%s]: %s\n", fieldName, detail)
		}
		return nil, fmt.Errorf("配置校验不通过")
	}
	// 高阶语义校验：检查 Watchlist 里的币，是否在 Tokens 字典里配了
	if err := checkSemanticLogic(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// checkSemanticLogic 负责业务逻辑上的关联校验
func checkSemanticLogic(cfg *Root) error {
	// 先把定义好的所有 network name 存进一个 map，方便 O(1) 查询
	definedNetworks := make(map[string]struct{})
	for name := range cfg.Networks {
		definedNetworks[name] = struct{}{}
	}

	// 遍历每一个 Watchlist 任务
	for _, wl := range cfg.Watchlists {
		for _, netName := range wl.Networks {
			// 核心逻辑：如果在 definedNetworks 里找不到这个名字，说明填错了
			if _, exists := definedNetworks[netName]; !exists {
				return fmt.Errorf(
					"❌ 配置文件错误: 任务 [%s] 使用了未定义的网络 [%s]。请检查 'networks' 块中是否声明了该网络",
					wl.Name, netName,
				)
			}
		}

		// 进阶校验：顺便检查 assets 里的 token 是否在 tokens 字典里（除了 native）
		for _, asset := range wl.Assets {
			if asset.Token == "native" {
				continue
			}
			if _, exists := cfg.Tokens[asset.Token]; !exists {
				return fmt.Errorf(
					"❌ 配置文件错误: 任务 [%s] 要求扫描代币 [%s]，但 'tokens' 字典中未配置该代币信息",
					wl.Name, asset.Token,
				)
			}
		}
	}

	return nil
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
