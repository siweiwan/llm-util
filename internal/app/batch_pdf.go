package app

import (
	"fmt"
	"llm-util/tui"
	"log/slog"
	path "path/filepath"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunModeB(poolSize int, xlsxFile string, progress chan<- tui.ProgressMsg) error {
	slog.Info("RunModeB start", "file", xlsxFile, "poolSize", poolSize)
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 20 {
		poolSize = 20
	}

	file, err := excelize.OpenFile(xlsxFile)
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

	// Read directory path from G1 (stored by template generator)
	fileDir, _ := file.GetCellValue("Sheet1", "G1")

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

		if len(row) < 2 {
			continue
		}

		request := row[0]
		fileName := row[1]

		if request == "" || fileName == "" {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: fileName, Status: "skip"}
			continue
		}

		// Construct full file path: dir from G1 + fileName from B column
		filePath := fileName
		if fileDir != "" {
			filePath = path.Join(fileDir, fileName)
		}

		// Skip if status column (D) already has value (断点续传)
		if len(row) >= 4 && row[3] != "" {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: request, Status: "skip"}
			continue
		}

		progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: request, Status: "processing"}

		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			rowIdx := i
			req := request
			fp := filePath

			resp, err := a.SendRequestWithFile(req, fp)

			now := time.Now().Format("2006-01-02 15:04:05")
			mu.Lock()
			if err != nil {
				slog.Error("RunModeB task failed", "row", rowIdx, "prompt", req, "file", fp, "err", err)
				file.SetCellValue("Sheet1", fmt.Sprintf("D%d", rowIdx+1), "失败")
				file.SetCellValue("Sheet1", fmt.Sprintf("F%d", rowIdx+1), err.Error())
				saveCounter++
			} else {
				slog.Info("RunModeB task done", "row", rowIdx, "prompt", req, "file", fp, "response_len", len(resp))
				file.SetCellValue("Sheet1", fmt.Sprintf("C%d", rowIdx+1), resp)
				file.SetCellValue("Sheet1", fmt.Sprintf("D%d", rowIdx+1), "完成")
				file.SetCellValue("Sheet1", fmt.Sprintf("E%d", rowIdx+1), now)
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
