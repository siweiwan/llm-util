package app

import (
	"fmt"
	"llm-util/constant"
	"llm-util/tui"
	"llm-util/util"
	"llm-util/util/console"
	"llm-util/util/file"
	"os"
	fpath "path/filepath"
	"strings"
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
	files, err := file.GetFiles(dir+"/pdfs", "pdf")
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 在 pdfs 目录下检索到 %d 个 PDF 文件", len(files)), constant.Green)
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
				if progress == nil {
					console.Colorful("📋 检测到未完成的进度文件，将继续处理...", constant.Yellow)
				}

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
				if progress == nil {
					console.Colorful("⚠️  检测到同名文件但问题不同，将创建新文件", constant.Yellow)
				}
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
		if err := f.SaveAs(filename); err != nil {
			if progress == nil {
				console.Colorful(fmt.Sprintf("❌ 保存Excel文件失败: %v", err), constant.Red)
			}
		} else if progress == nil {
			console.Colorful(fmt.Sprintf("\n✅ 所有结果已保存至: %s", filename), constant.Green)
		}
		if err := f.Close(); err != nil {
			if progress == nil {
				console.Colorful(fmt.Sprintf("❌ 关闭Excel文件失败: %v", err), constant.Red)
			}
		}
	}()

	var pendingFiles []string
	for _, filePath := range files {
		md5, err := file.CalculateMD5(filePath)
		if err == nil && processedMD5[md5] {
			if progress == nil {
				console.Colorful(fmt.Sprintf("⏭️  跳过已处理文件: %s", fpath.Base(filePath)), constant.Yellow)
			}
			continue
		}
		pendingFiles = append(pendingFiles, filePath)
	}

	if progress == nil {
		fmt.Println("\n" + strings.Repeat("-", 80))
		console.Colorful(fmt.Sprintf("📊 需要新处理的文件数量/总文件数量: %d/%d", len(pendingFiles), len(files)), constant.Blue)
		fmt.Println(strings.Repeat("-", 80))
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

		if progress != nil {
			progress <- tui.ProgressMsg{Index: i, Total: totalFiles, Filename: fpath.Base(filePath), Status: "processing"}
		}

		pool.Submit(func() {
			defer wg.Done()

			if progress == nil {
				console.Colorful(fmt.Sprintf("\n🔄 [%d/%d] 正在处理: %s", i+1, len(pendingFiles), fpath.Base(filePath)), constant.Blue)
			}

			answer, err := a.SendRequestWithFile(input, filePath)
			if err != nil {
				if progress != nil {
					progress <- tui.ProgressMsg{Index: i, Total: totalFiles, Filename: fpath.Base(filePath), Status: "error"}
				} else {
					console.Colorful(fmt.Sprintf("❌ 文件[%s] 请求失败: %v", fpath.Base(filePath), err), constant.Red)
				}
				return
			}
			if progress == nil {
				console.Colorful(fmt.Sprintf("✅ [%d/%d] %s 处理完成", i+1, len(pendingFiles), fpath.Base(filePath)), constant.Green)
				fmt.Printf(constant.Green+"\n📄 %s 回答内容:\n%s\n"+constant.Reset, fpath.Base(filePath), answer)
			}

			fileName := file.RemoveFileExtension(fpath.Base(filePath))
			md5, _ := file.CalculateMD5(filePath)

			data := []string{fileName, md5, answer}

			for col, value := range data {
				cell, _ := excelize.CoordinatesToCellName(col+1, currentRow)
				if err := f.SetCellValue("Sheet1", cell, value); err != nil {
					if progress == nil {
						console.Colorful(fmt.Sprintf("⚠️  写入数据到%s失败: %v", cell, err), constant.Yellow)
					}
				}
			}

			mu.Lock()
			currentRow++
			if err := f.SaveAs(filename); err != nil {
				if progress == nil {
					console.Colorful(fmt.Sprintf("⚠️  临时保存失败: %v", err), constant.Yellow)
				}
			}
			mu.Unlock()

			if progress != nil {
				progress <- tui.ProgressMsg{Index: i, Total: totalFiles, Filename: fpath.Base(filePath), Status: "done"}
			}
		})
	}
	wg.Wait()
	return nil
}
