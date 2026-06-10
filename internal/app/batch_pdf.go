package app

import (
	"fmt"
	"llm-util/tui"
	"llm-util/util"
	"llm-util/util/file"
	"os"
	fpath "path/filepath"
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/xuri/excelize/v2"
)

func (a *App) RunPdfBatchQuery(poolSize int, question string, progress chan<- tui.ProgressMsg) error {
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}
	files, err := file.GetFiles(dir+"/pdfs", "pdf")
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	input := question

	filename := util.GetFirstXChars(input, 20) + ".xlsx"
	processedMD5 := make(map[string]bool)
	var f *excelize.File
	currentRow := 3

	if _, err := os.Stat(filename); err == nil {
		existingFile, err := excelize.OpenFile(filename)
		if err == nil {
			existingQuestion, _ := existingFile.GetCellValue("Sheet1", "A1")
			if existingQuestion == input {
				f = existingFile

				rows, _ := existingFile.GetRows("Sheet1")
				for rowIdx, row := range rows {
					if rowIdx < 2 {
						continue
					}
					if len(row) >= 2 {
						processedMD5[row[1]] = true
					}
				}
				currentRow = len(rows) + 1
			} else {
				filename = util.GetFirstXChars(input, 20) + "_new" + ".xlsx"
			}
		}
	}

	if f == nil {
		f = excelize.NewFile()
		f.SetCellValue("Sheet1", "A1", input)
		headers := []string{"文件名", "MD5", "回答内容"}
		for col, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(col+1, 2)
			f.SetCellValue("Sheet1", cell, header)
		}
	}

	defer func() {
		_ = f.SaveAs(filename)
		_ = f.Close()
	}()

	var pendingFiles []string
	for _, filePath := range files {
		md5, err := file.CalculateMD5(filePath)
		if err == nil && processedMD5[md5] {
			continue
		}
		pendingFiles = append(pendingFiles, filePath)
	}

	totalFiles := len(pendingFiles)

	var wg sync.WaitGroup
	var mu sync.Mutex

	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	for i, filePath := range pendingFiles {
		wg.Add(1)
		filePath := filePath
		i := i

		progress <- tui.ProgressMsg{Index: i, Total: totalFiles, Filename: fpath.Base(filePath), Status: "processing"}

		pool.Submit(func() {
			defer wg.Done()

			answer, err := a.SendRequestWithFile(input, filePath)
			if err != nil {
				progress <- tui.ProgressMsg{Index: i, Total: totalFiles, Filename: fpath.Base(filePath), Status: "error"}
				return
			}

			fileName := file.RemoveFileExtension(fpath.Base(filePath))
			md5, _ := file.CalculateMD5(filePath)

			data := []string{fileName, md5, answer}

			for col, value := range data {
				cell, _ := excelize.CoordinatesToCellName(col+1, currentRow)
				_ = f.SetCellValue("Sheet1", cell, value)
			}

			mu.Lock()
			currentRow++
			_ = f.SaveAs(filename)
			mu.Unlock()

			progress <- tui.ProgressMsg{Index: i, Total: totalFiles, Filename: fpath.Base(filePath), Status: "done"}
		})
	}
	wg.Wait()
	return nil
}
