package app

import (
	"context"
	"fmt"
	"llm-util/ai/app/impl/bailian"
	"llm-util/tui"
	"log/slog"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunModeA(poolSize int, filename string, progress chan<- tui.ProgressMsg) error {
	slog.Info("RunModeA start", "file", filename, "poolSize", poolSize)
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 20 {
		poolSize = 20
	}

	file, err := excelize.OpenFile(filename)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	rows, err := file.GetRows("Sheet1")
	if err != nil {
		return fmt.Errorf("读取行数据失败: %w", err)
	}
	if len(rows) < 2 {
		return fmt.Errorf("至少需要一行标题和一行数据")
	}

	client := bailian.NewClientWithAppIDAPIKey(a.AppId, a.APIKey)
	totalRows := len(rows) - 1

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	saveCounter := 0
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) == 0 || row[0] == "" {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Status: "skip"}
			continue
		}
		prompt := row[0]
		if len(row) >= 3 && row[2] == "完成" {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: prompt, Status: "skip"}
			continue
		}

		progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: prompt, Status: "processing"}

		wg.Add(1)
		rowIdx := i
		req := prompt
		pool.Submit(func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("SDK panic: %v", r)
					slog.Error("RunModeA SDK panic recovered", "row", rowIdx, "err", err)
					mu.Lock()
					file.SetCellValue("Sheet1", fmt.Sprintf("C%d", rowIdx+1), "失败")
					file.SetCellValue("Sheet1", fmt.Sprintf("E%d", rowIdx+1), err.Error())
					saveCounter++
					if saveCounter >= 10 {
						_ = file.Save()
						saveCounter = 0
					}
					mu.Unlock()
					progress <- tui.ProgressMsg{Index: rowIdx, Total: totalRows, Filename: req, Status: "error"}
				}
			}()

			resp, err := client.CreateChatCompletion(context.Background(), bailian.ChatCompletionRequest{
				Input: &bailian.RequestInput{Prompt: req},
			})

			now := time.Now().Format("2006-01-02 15:04:05")
			mu.Lock()
			if err != nil {
				slog.Error("RunModeA task failed", "row", rowIdx, "prompt", req, "err", err)
				file.SetCellValue("Sheet1", fmt.Sprintf("C%d", rowIdx+1), "失败")
				file.SetCellValue("Sheet1", fmt.Sprintf("E%d", rowIdx+1), err.Error())
				saveCounter++
			} else {
				slog.Info("RunModeA task done", "row", rowIdx, "prompt", req, "response_len", len(resp.Output.Text))
				file.SetCellValue("Sheet1", fmt.Sprintf("B%d", rowIdx+1), resp.Output.Text)
				file.SetCellValue("Sheet1", fmt.Sprintf("C%d", rowIdx+1), "完成")
				file.SetCellValue("Sheet1", fmt.Sprintf("D%d", rowIdx+1), now)
				saveCounter++
			}
			if saveCounter >= 10 {
				_ = file.Save()
				saveCounter = 0
			}
			mu.Unlock()
			if err != nil {
				progress <- tui.ProgressMsg{Index: rowIdx, Total: totalRows, Filename: req, Status: "error"}
			} else {
				progress <- tui.ProgressMsg{Index: rowIdx, Total: totalRows, Filename: req, Status: "done"}
			}
		})
	}

	wg.Wait()
	return file.Save()
}
