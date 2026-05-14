package app

import (
	"context"
	"fmt"
	"llm-util/ai/app/impl/bailian"
	"llm-util/constant"
	"llm-util/tui"
	"llm-util/util/console"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunWorkflowQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
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
		fmt.Println("\n📂 正在打开 workflow.xlsx...")
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
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 成功读取 %d 行数据", len(rows)), constant.Green)
	}

	head := rows[0]

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	var errCount int
	totalRows := len(rows) - 1

	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) == 0 {
			continue
		}
		question := row[0]

		if len(row) >= 2 && row[1] != "" {
			if progress != nil {
				progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "skip"}
			}
			continue
		}

		iCopy := i
		rowCopy := append([]string{}, row...)

		if progress == nil {
			PrintQuestion(question)
		} else {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "processing"}
		}

		wg.Add(1)
		_ = pool.Submit(func() {
			defer wg.Done()

			argsM := make(map[string]string)
			for col := 2; col < len(head) && col < len(rowCopy); col++ {
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
				if progress != nil {
					progress <- tui.ProgressMsg{Index: iCopy, Total: totalRows, Filename: question, Status: "error"}
				} else {
					console.Colorful(fmt.Sprintf("❌ 请求失败: %v", err), constant.Red)
				}
				mu.Lock()
				errCount++
				mu.Unlock()
				return
			}

			if progress == nil {
				console.Colorful(fmt.Sprintf("✅ 问题 [%d] 处理完成", iCopy+1), constant.Green)
			}

			mu.Lock()
			file.SetCellValue("Sheet1", fmt.Sprintf("B%d", iCopy+1), response.Output.Text)
			mu.Unlock()

			if progress != nil {
				progress <- tui.ProgressMsg{Index: iCopy, Total: totalRows, Filename: question, Status: "done"}
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
		console.Colorful(fmt.Sprintf("✅ 规则模式【工作流】处理完毕！耗时: %v", time.Since(start)), constant.Yellow)
		fmt.Println(strings.Repeat("=", 80))
		if errCount == 0 {
			console.Colorful("🎉🎉🎉 所有请求成功完成！", constant.Green)
		} else {
			console.Colorful(fmt.Sprintf("⚠️  请求失败数量: %d", errCount), constant.Red)
		}
	}
	return nil
}
