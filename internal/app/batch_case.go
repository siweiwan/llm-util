package app

import (
	"fmt"
	"llm-util/constant"
	"llm-util/tui"
	"llm-util/util/console"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunCaseQueryRule(poolSize int, progress chan<- tui.ProgressMsg) {
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), constant.Green)
	}

	start := time.Now()

	if progress == nil {
		fmt.Println("\n📂 正在打开 data.xlsx...")
	}
	file, err := excelize.OpenFile("data.xlsx")
	if err != nil {
		if progress == nil {
			console.Colorful(fmt.Sprintf("❌ 打开文件失败: %v", err), constant.Red)
		}
		return
	}
	defer file.Close()

	rows, err := file.GetRows("Sheet1")
	if err != nil {
		if progress == nil {
			console.Colorful(fmt.Sprintf("❌ 读取行数据失败: %v", err), constant.Red)
		}
		return
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 成功读取 %d 行数据", len(rows)), constant.Green)
	}

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	var (
		cacheChat string
		errCount  int
	)

	totalRows := len(rows) - 1

	for i, row := range rows {
		if i == 0 {
			continue
		}

		question := row[0]
		if len(row) >= 2 {
			if row[1] != "" {
				if progress != nil {
					progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "skip"}
				}
				continue
			}
		}

		if progress == nil {
			PrintQuestion(question)
		} else {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "processing"}
		}

		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			i := i
			question := question

			response, err := a.SendRequest(question)
			if err != nil {
				if progress != nil {
					progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "error"}
				} else {
					fmt.Printf("请求失败: %v\n", err)
				}
				errCount++
				return
			}

			mu.Lock()
			cacheChat += response
			file.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+1), response)
			mu.Unlock()

			if progress != nil {
				progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "done"}
			}
		})
	}

	wg.Wait()

	if progress == nil {
		fmt.Println("\n💾 正在保存 Excel 文件...")
	}
	if err := file.Save(); err != nil {
		if progress == nil {
			console.Colorful(fmt.Sprintf("❌ 保存文件失败: %v", err), constant.Red)
		}
	}

	if progress == nil {
		fmt.Println("\n" + strings.Repeat("=", 80))
		console.Colorful(fmt.Sprintf("✅ 规则模式【案例查询】处理完毕！耗时: %v", time.Since(start)), constant.Yellow)
		fmt.Println(strings.Repeat("=", 80))
		if errCount == 0 {
			console.Colorful("🎉🎉🎉 所有请求成功完成！", constant.Green)
		} else {
			console.Colorful(fmt.Sprintf("⚠️  请求失败数量: %d", errCount), constant.Red)
		}
	}
}
