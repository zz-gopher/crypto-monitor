package db

import (
	"crypto-monitor/tools"
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AssetRecord 状态表 (State Table)：只存最新快照，供线上极速查询
type AssetRecord struct {
	gorm.Model
	WalletAddress string `gorm:"uniqueIndex:idx_asset_key;size:42"` // 钱包地址
	TokenContract string `gorm:"uniqueIndex:idx_asset_key;size:42"` // 合约地址
	ChainID       int    `gorm:"uniqueIndex:idx_asset_key"`         // 链 ID (例如: 1, 56)
	TokenSymbol   string `gorm:"index"`                             // 符号加个普通索引，方便有时候按名称模糊查
	ChainName     string // 链名称
	Decimals      uint8  // 代币精度
	Balance       string // 余额用 string 防精度丢失
	BlockHeight   uint64 // 最新更新的区块高度
}

// AssetHistoryLog 流水表 (History Log Table)：只追加不修改，存历史轨迹
type AssetHistoryLog struct {
	gorm.Model           // 自带的 CreatedAt 字段完美充当 "扫描时间戳"
	WalletAddress string `gorm:"index;size:42"`
	TokenContract string `gorm:"index;size:42"`
	ChainID       int    `gorm:"index"`
	TokenSymbol   string
	ChainName     string
	Decimals      uint8 // 代币精度
	Balance       string
	BlockHeight   uint64
}

// InitDB 初始化SqlLite
func InitDB() *gorm.DB {
	// 自动在本地生成一个名为 crypto_monitor.db 的文件
	db, err := gorm.Open(sqlite.Open("crypto_monitor.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	// 自动迁移：自动根据上面的结构体创建表和索引！
	fmt.Println("正在初始化多链双表架构...")
	_ = db.AutoMigrate(&AssetRecord{}, &AssetHistoryLog{})
	return db
}

// StartDataProcessor 事务性写入
func StartDataProcessor(db *gorm.DB, dataChan <-chan AssetRecord, exporter *tools.CSVExporter, enableCSV bool) {
	var batch []AssetRecord
	batchSize := 100 // 满 100 条触发批处理
	ticker := time.NewTicker(2 * time.Second)
	processBatch := func() {
		if len(batch) == 0 {
			return
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			// Upsert 更新状态表
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "wallet_address"}, {Name: "token_contract"}, {Name: "chain_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"balance", "block_height", "updated_at"}),
			}).CreateInBatches(batch, len(batch)).Error
			if err != nil {
				return err
			} // 报错立刻回滚

			// 组装并追加流水表
			var historyBatch []AssetHistoryLog
			for _, b := range batch {
				historyBatch = append(historyBatch, AssetHistoryLog{
					WalletAddress: b.WalletAddress,
					TokenContract: b.TokenContract,
					ChainID:       b.ChainID,
					TokenSymbol:   b.TokenSymbol,
					ChainName:     b.ChainName,
					Decimals:      b.Decimals,
					Balance:       b.Balance,
					BlockHeight:   b.BlockHeight,
				})
			}
			err = tx.CreateInBatches(historyBatch, len(historyBatch)).Error
			if err != nil {
				return err
			} // 报错连带状态表一起回滚

			return nil // 完美提交
		})
		if err != nil {
			fmt.Printf("[Error] 数据库落盘失败，触发一致性回滚: %v\n", err)
		} else {
			if enableCSV && exporter != nil {
				var csvBatch [][]string
				currentTime := time.Now().Format("2006-01-02 15:04:05")
				for _, b := range batch {
					csvBatch = append(csvBatch, []string{
						// 确保这里的顺序和你的 exportHeaders 一模一样
						b.ChainName,                      // 链(Chain)
						b.TokenContract,                  // 代币地址(Token)
						b.WalletAddress,                  // 钱包地址(Owner)
						b.TokenSymbol,                    // 代币名称(Symbol)
						b.Balance,                        // 余额(Balance)
						fmt.Sprintf("%d", b.Decimals),    // 精度
						currentTime,                      //创建时间
						fmt.Sprintf("%d", b.BlockHeight), // 区块高度
						"true",                           // 是否成功 (进到这里的都是成功的)
					})
				}
				// 一次性批量写入并刷盘，只加一次锁！
				if writeErr := exporter.WriteBatch(csvBatch); writeErr != nil {
					fmt.Printf("[Warn] 旁路降级：数据已存入DB，但追加CSV失败: %v\n", writeErr)
				}
			}
		}
		// 清空缓冲区，准备装载下一批
		batch = batch[:0]
	}
	// 持续监听 Channel
	for {
		select {
		case record, ok := <-dataChan:
			if !ok {
				processBatch() // 通道关闭，处理最后一波残留数据
				return
			}
			batch = append(batch, record)
			if len(batch) >= batchSize {
				processBatch()
			}
		case <-ticker.C:
			// 时间到了强制刷入硬盘，防止在内存长时间滞留
			processBatch()
		}
	}
}
