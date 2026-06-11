package app

import (
	"fmt"
	"llm-util/tui"
	"llm-util/util/file"
	"log/slog"
	"os"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunDIYQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
	slog.Info("RunDIYQueryRule start", "poolSize", poolSize)
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}
	_, err = file.GetFiles(dir+"/files", "")
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
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
		} else if len(row) > 3 {
			if row[3] == "完成" {
				progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: row[1], Status: "skip"}
				continue
			}
		} else {
			if row[0] == "" || row[1] == "" {
				progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: row[1], Status: "skip"}
				continue
			}
		}

		question := row[0]
		fileName := row[1]

		progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "processing"}

		wg.Add(1)
		pool.Submit(func() {
			i := i
			input := question
			filePath := dir + "/files/" + fileName

			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("SDK panic: %v", r)
					slog.Error("RunDIY SDK panic recovered", "row", i, "err", err)
					mu.Lock()
					excelFile.SetCellValue("Sheet1", fmt.Sprintf("D%d", i+1), "失败")
					excelFile.SetCellValue("Sheet1", fmt.Sprintf("E%d", i+1), err.Error())
					_ = excelFile.Save()
					mu.Unlock()
					progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "error"}
				}
			}()

			answer, err := a.SendRequestWithFile(input, filePath)
			if err != nil {
				slog.Error("RunDIYQueryRule task failed", "row", i, "file", fileName, "err", err)
				mu.Lock()
				excelFile.SetCellValue("Sheet1", fmt.Sprintf("D%d", i+1), "失败")
				excelFile.SetCellValue("Sheet1", fmt.Sprintf("E%d", i+1), err.Error())
				_ = excelFile.Save()
				mu.Unlock()
				progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "error"}
				return
			}
			slog.Info("RunDIYQueryRule task done", "row", i, "file", fileName, "response_len", len(answer))

			mu.Lock()
			excelFile.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+1), answer)
			excelFile.SetCellValue("Sheet1", fmt.Sprintf("D%d", i+1), "完成")
			_ = excelFile.Save()
			mu.Unlock()

			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "done"}
		})
	}
	wg.Wait()
	return nil
}
