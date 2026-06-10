package app

import (
	"context"
	"fmt"
	"llm-util/ai/app/impl/bailian"
	"llm-util/tui"
	"log/slog"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunWorkflowQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
	slog.Info("RunWorkflowQueryRule start", "poolSize", poolSize)
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}

	file, err := excelize.OpenFile("workflow.xlsx")
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	rows, err := file.GetRows("Sheet1")
	if err != nil {
		return fmt.Errorf("读取行数据失败: %w", err)
	}
	if len(rows) < 2 {
		return fmt.Errorf("Excel中没有数据")
	}

	head := rows[0]

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	totalRows := len(rows) - 1

	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) == 0 {
			continue
		}
		question := row[0]

		if len(row) >= 3 && row[2] == "完成" {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "skip"}
			continue
		}

		iCopy := i
		rowCopy := append([]string{}, row...)

		progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "processing"}

		wg.Add(1)
		_ = pool.Submit(func() {
			defer wg.Done()

			argsM := make(map[string]string)
			for col := 3; col < len(head) && col < len(rowCopy); col++ {
				argsM[head[col]] = rowCopy[col]
			}

			client := bailian.NewClientWithAppIDAPIKey(a.AppId, a.APIKey)
			response, err := client.CreateChatCompletion(context.TODO(), bailian.ChatCompletionRequest{
				Input: &bailian.RequestInput{
					Prompt:    question,
					BizParams: argsM,
				},
			})
			if err != nil {
				slog.Error("RunWorkflowQueryRule task failed", "row", iCopy, "prompt", question, "err", err)
				mu.Lock()
				file.SetCellValue("Sheet1", fmt.Sprintf("C%d", iCopy+1), "失败")
				mu.Unlock()
				progress <- tui.ProgressMsg{Index: iCopy, Total: totalRows, Filename: question, Status: "error"}
				return
			}

			slog.Info("RunWorkflowQueryRule task done", "row", iCopy, "prompt", question, "response_len", len(response.Output.Text))
			mu.Lock()
			file.SetCellValue("Sheet1", fmt.Sprintf("B%d", iCopy+1), response.Output.Text)
			file.SetCellValue("Sheet1", fmt.Sprintf("C%d", iCopy+1), "完成")
			mu.Unlock()

			progress <- tui.ProgressMsg{Index: iCopy, Total: totalRows, Filename: question, Status: "done"}
		})
	}

	wg.Wait()

	_ = file.Save()
	return nil
}
