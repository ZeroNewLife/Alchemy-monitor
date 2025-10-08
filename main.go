package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Конфигурация
var config = struct {
	MyAddresses      []string // Твои адреса для отслеживания
	TrackOnlyMine    bool     // Только мои транзакции
	LogToFile        bool
	LogFileName      string
}{
	MyAddresses: []string{
		"0xD843CBe0bdeE3E884Fd32cE4942219830D5944DA", // твой адрес
	},
	TrackOnlyMine: true, // ✅ ВКЛЮЧАЕМ ФИЛЬТР ТОЛЬКО МОИХ ТРАНЗАКЦИЙ
	LogToFile:     true,
	LogFileName:   "my_transactions.log",
}

func main() {
	// Открываем файл для логов
	var logFile *os.File
	var err error
	if config.LogToFile {
		logFile, err = os.OpenFile(config.LogFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("Ошибка открытия файла логов:", err)
		}
		defer logFile.Close()
	}

	// Функция для логирования
	logMessage := func(format string, args ...interface{}) {
		message := fmt.Sprintf(format, args...)
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fullMessage := fmt.Sprintf("[%s] %s\n", timestamp, message)
		
		fmt.Print(fullMessage)
		if config.LogToFile {
			if _, err := logFile.WriteString(fullMessage); err != nil {
				fmt.Printf("Ошибка записи в файл: %v\n", err)
			}
		}
	}

	client, err := ethclient.Dial("https://eth-sepolia.g.alchemy.com/v2/Your api key ")
	if err != nil {
		logMessage("❌ Ошибка подключения: %v", err)
		log.Fatal(err)
	}
	defer client.Close()

	logMessage("🚀 Мониторинг Sepolia запущен (ТОЛЬКО мои транзакции)")
	logMessage("📡 Отслеживаю адрес: %s", config.MyAddresses[0])
	logMessage("💾 Логи в файле: %s", config.LogFileName)

	lastBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		logMessage("❌ Ошибка получения блока: %v", err)
		log.Fatal(err)
	}

	for {
		currentBlock, err := client.BlockNumber(context.Background())
		if err != nil {
			logMessage("⚠️  Ошибка получения номера блока: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if currentBlock > lastBlock {
			myTransactionsFound := 0
			
			for blockNum := lastBlock + 1; blockNum <= currentBlock; blockNum++ {
				block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(blockNum)))
				if err != nil {
					if err.Error() == "transaction type not supported" {
						continue
					}
					logMessage("❌ Ошибка получения блока #%d: %v", blockNum, err)
					continue
				}

				// Проверяем все транзакции в блоке
				for _, tx := range block.Transactions() {
					if isMyTransaction(tx) {
						myTransactionsFound++
						logTransaction(tx, block.Number().Uint64(), logMessage)
					}
				}
			}

			if myTransactionsFound > 0 {
				logMessage("📊 В блоках #%d-#%d найдено моих транзакций: %d", 
					lastBlock+1, currentBlock, myTransactionsFound)
			} else {
				logMessage("👀 Новые блоки #%d-#%d (моих транзакций нет)", 
					lastBlock+1, currentBlock)
			}
			
			lastBlock = currentBlock
		}

		time.Sleep(12 * time.Second)
	}
}

// Проверяем принадлежит ли транзакция мне
func isMyTransaction(tx *types.Transaction) bool {
    from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
    if err != nil {
        return false
    }

    fromStr := from.Hex()
    
    // Проверяем отправителя
    for _, myAddr := range config.MyAddresses {
        if fromStr == myAddr {
            return true  // ✅ Любая транзакция ОТ меня
        }
    }

    return false
}

// Логируем информацию о транзакции
func logTransaction(tx *types.Transaction, blockNumber uint64, logMessage func(string, ...interface{})) {
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	
	transactionType := "OUTGOING"
	for _, myAddr := range config.MyAddresses {
		if from.Hex() == myAddr {
			transactionType = "OUTGOING 📤"
		} else if tx.To().Hex() == myAddr {
			transactionType = "INCOMING 📥"
		}
	}

	logMessage("🎯 МОЯ ТРАНЗАКЦИЯ [%s]", transactionType)
	logMessage("   ├─ Блок: #%d", blockNumber)
	logMessage("   ├─ От: %s", from.Hex())
	logMessage("   ├─ Кому: %s", tx.To().Hex())
	logMessage("   ├─ Сумма: %s ETH", formatEther(tx.Value()))
	logMessage("   ├─ Gas: %d", tx.Gas())
	logMessage("   └─ Hash: %s", tx.Hash().Hex())
}

// Конвертируем wei в ETH
func formatEther(wei *big.Int) string {
	eth := new(big.Float).SetInt(wei)
	eth = eth.Quo(eth, big.NewFloat(1e18))
	return eth.Text('f', 6)
}