package app

import (
	"fmt"
	"llm-util/constant"
	"llm-util/tui"
	"llm-util/util/console"
	"llm-util/util/file"
	"os"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunDIYQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), constant.Green)
	}

	if progress == nil {
		fmt.Println("\n🔍 文件检索中...")
	}
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}
	files, err := file.GetFiles(dir+"/files", "")
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 在 files 目录下检索到 %d 个文件", len(files)), constant.Green)
	}

	if progress == nil {
		fmt.Println("\n📂 正在打开 process.xlsx...")
	}
	excelFile, err := excelize.OpenFile("process.xlsx")
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer excelFile.Close()

	rows, err := excelFile.GetRows("Sheet1")
	if err != nil {
		return fmt.Errorf("读取行数据失败: %w", err)
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 成功读取 %d 行数据", len(rows)), constant.Green)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	totalRows := len(rows) - 1

	defer excelFile.Save()
	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) < 2 {
			continue
		} else if len(row) > 2 {
			if row[2] != "" {
				if progress != nil {
					progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: row[1], Status: "skip"}
				}
				continue
			}
		} else {
			if row[0] == "" || row[1] == "" {
				continue
			}
		}

		question := row[0]
		fileName := row[1]

		if progress != nil {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "processing"}
		}

		wg.Add(1)
		pool.Submit(func() {
			i := i
			input := question
			filePath := dir + "/files/" + fileName

			defer wg.Done()

			if progress == nil {
				console.Colorful(fmt.Sprintf("\n🔄 正在处理 %s", fileName), constant.Blue)
				console.Colorful(fmt.Sprintf("   问题: %s", input), constant.Blue)
			}

			answer, err := a.SendRequestWithFile(input, filePath)
			if err != nil {
				if progress != nil {
					progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "error"}
				} else {
					console.Colorful(fmt.Sprintf("❌ 文件[%s] 请求失败: %v", fileName, err), constant.Red)
				}
				return
			}
			if progress == nil {
				console.Colorful(fmt.Sprintf("✅ %s 处理完成", fileName), constant.Green)
				fmt.Printf(constant.Green+"\n📄 %s 回答内容:\n%s\n"+constant.Reset, fileName, answer)
			}

			mu.Lock()
			excelFile.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+1), answer)
			excelFile.Save()
			mu.Unlock()

			if progress != nil {
				progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "done"}
			}
		})
	}
	wg.Wait()
	return nil
}
