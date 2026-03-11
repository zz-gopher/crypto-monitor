package metadata

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Cache 是全局唯一的 Token 元数据缓存中心
type Cache struct {
	dir      string
	ttl      time.Duration
	memCache sync.Map
	sf       singleflight.Group
}

// NewCache 构造函数
func NewCache(dir string, ttl time.Duration) *Cache {
	// 启动时确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[Warning] 无法创建缓存目录: %v", err)
	}
	return &Cache{dir: dir, ttl: ttl}
}

// LoadFromDisk 启动预热 把本地的JSON吸入内存
func (c *Cache) LoadFromDisk() {
	files, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	count := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(c.dir, file.Name()))
		if err != nil {
			continue
		}
		var meta TokenMetadata
		if err := json.Unmarshal(data, &meta); err == nil {
			key := fmt.Sprintf("%s_%s", meta.ChainID, meta.Address)
			c.memCache.Store(key, &meta)
			count++
		}
	}
	fmt.Printf("🚀 成功从磁盘预热加载了 %d 个 Token 的元数据", count)
}

// GetOrFetch 核心拦截器
func (c *Cache) GetOrFetch(chainID, address string, fetchRPC func() (*TokenMetadata, error)) (*TokenMetadata, error) {
	key := fmt.Sprintf("%s_%s", chainID, address)
	// 第一层去查内存
	if val, ok := c.memCache.Load(key); ok {
		meta := val.(*TokenMetadata)
		// 校验保质期
		if time.Now().Unix()-meta.CachedAt < int64(c.ttl.Seconds()) {
			return meta, nil // 命中，0毫秒返回！
		}
	}
	// 第二层防击穿排队（同一个key瞬间只有一个协程能真正执行fetchRPC）
	v, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// 去链上查询
		newMeta, err := fetchRPC()
		if err != nil {
			return nil, err
		}
		// 补全信息
		newMeta.ChainID = chainID
		newMeta.Address = address
		newMeta.CachedAt = time.Now().Unix()

		// 存入内存
		c.memCache.Store(key, newMeta)
		// 开启极轻量协程异步落盘，不卡主线程
		go c.saveToDisk(key, newMeta)

		return newMeta, nil
	})

	if err != nil {
		return nil, err
	}
	return v.(*TokenMetadata), nil
}

// 生成json文件，存入硬盘
func (c *Cache) saveToDisk(key string, meta *TokenMetadata) {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err == nil {
		_ = os.WriteFile(filepath.Join(c.dir, key+".json"), data, 0644)
	}
}
