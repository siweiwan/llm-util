package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llm-util/ai/app/impl/bailian"
	"llm-util/constant"
	"llm-util/file/qwen"
	"llm-util/tui"
	"llm-util/util"
	"llm-util/util/console"
	"llm-util/util/file"
	"net/http"
	"os"
	fpath "path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/panjf2000/ants/v2"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

var apiKey = ""
var appId = ""

var conversationHistory []Message

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	_ = godotenv.Load()
	ensureEnvFile()

	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	if appId == "" {
		appId = os.Getenv("LLM_APP_ID")
	}

	model := tui.NewModel(apiKey, appId)
	model.OnSaveSettings = func(key, id string) error {
		apiKey = key
		appId = id
		return saveEnvFile(key, id)
	}
	model.OnSend = func(prompt string, history []tui.Message) (string, error) {
		conversationHistory = nil
		for _, m := range history {
			conversationHistory = append(conversationHistory, Message{Role: m.Role, Content: m.Content})
		}
		return sendRequest(prompt)
	}
	model.OnSendFile = func(prompt, filePath string) (string, error) {
		return sendRequestWithFile(prompt, filePath)
	}
	model.OnRunCase = func(poolSize int, progress chan<- tui.ProgressMsg) error {
		runCaseQueryRule(poolSize, progress)
		return nil
	}
	model.OnRunPDF = func(poolSize int, question string, progress chan<- tui.ProgressMsg) error {
		return runPdfBatchQuery(poolSize, question, progress)
	}
	model.OnRunDIY = func(poolSize int, progress chan<- tui.ProgressMsg) error {
		return runDIYQueryRule(poolSize, progress)
	}
	model.OnRunWorkflow = func(poolSize int, progress chan<- tui.ProgressMsg) error {
		return runWorkflowQueryRule(poolSize, progress)
	}

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}


func ensureEnvFile() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		f, err := os.Create(".env")
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString("LLM_API_KEY=\nLLM_APP_ID=\nALIBABA_CLOUD_ACCESS_KEY_ID=\nALIBABA_CLOUD_ACCESS_KEY_SECRET=\n")
	}
}

func saveEnvFile(apiKey, appId string) error {
	data, err := os.ReadFile(".env")
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	updatedKey, updatedId := false, false
	for i, line := range lines {
		if strings.HasPrefix(line, "LLM_API_KEY=") {
			lines[i] = "LLM_API_KEY=" + apiKey
			updatedKey = true
		}
		if strings.HasPrefix(line, "LLM_APP_ID=") {
			lines[i] = "LLM_APP_ID=" + appId
			updatedId = true
		}
	}
	if !updatedKey {
		lines = append(lines, "LLM_API_KEY="+apiKey)
	}
	if !updatedId {
		lines = append(lines, "LLM_APP_ID="+appId)
	}
	return os.WriteFile(".env", []byte(strings.Join(lines, "\n")), 0644)
}

func sendRequest(prompt string) (string, error) {
	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/apps/%s/completion", appId)

	requestBody := map[string]interface{}{
		"input": map[string]interface{}{
			"prompt":   prompt,
			"messages": conversationHistory,
		},
		"parameters": map[string]interface{}{},
		"debug":      map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON编码失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	// 创建一个通道用于终止 loading 动画
	done := make(chan struct{})
	go showLoading(done)

	resp, err := client.Do(req)
	close(done)
	if err != nil {
		return "", fmt.Errorf("请求发送失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API请求失败，状态码: %d，响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response struct {
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("响应解析失败: %w", err)
	}

	return response.Output.Text, nil
}

func sendRequestWithFile(prompt, filePath string) (string, error) {
	addFileResponse, err := qwen.UploadFile(filePath)
	if err != nil {
		fmt.Printf("上传文件失败: %v\n", err)
		return "", err
	}
	sessionFileId := *addFileResponse.Body.Data.FileId

	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/apps/%s/completion", appId)

	requestBody := map[string]interface{}{
		"input": map[string]interface{}{
			"prompt":   prompt,
			"messages": conversationHistory,
		},
		"parameters": map[string]interface{}{
			"rag_options": map[string]interface{}{
				"session_file_ids": []string{sessionFileId},
			},
		},
		"debug": map[string]interface{}{},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON编码失败: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求发送失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API请求失败，状态码: %d，响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response struct {
		Output struct {
			Text string `json:"text"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("响应解析失败: %w", err)
	}

	return response.Output.Text, nil
}

func runCaseQueryRule(poolSize int, progress chan<- tui.ProgressMsg) {
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), constant.Green)
	}

	start := time.Now()

	// 读取Excel文件
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

	// 获取所有行数据
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

	totalRows := len(rows) - 1 // skip header

	for i, row := range rows {
		if i == 0 {
			continue // 跳过标题行
		}

		question := row[0]
		if len(row) >= 2 {
			if row[1] != "" {
				// 跳过已处理过的文件
				if progress != nil {
					progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "skip"}
				}
				continue
			}
		}

		if progress == nil {
			printQuestion(question)
		} else {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "processing"}
		}

		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			i := i
			question := question

			response, err := sendRequest(question)
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

// 示例：规则2 单问题批量提问
func runPdfBatchQuery(poolSize int, question string, progress chan<- tui.ProgressMsg) error {
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

	// 生成文件名
	filename := util.GetFirstXChars(input, 20) + ".xlsx"
	processedMD5 := make(map[string]bool)
	var f *excelize.File
	currentRow := 3

	// 检查已有文件
	if _, err := os.Stat(filename); err == nil {
		// 打开现有文件
		existingFile, err := excelize.OpenFile(filename)
		if err == nil {
			// 检查问题是否一致
			existingQuestion, _ := existingFile.GetCellValue("Sheet1", "A1")
			if existingQuestion == input {
				f = existingFile
				if progress == nil {
					console.Colorful("📋 检测到未完成的进度文件，将继续处理...", constant.Yellow)
				}

				// 读取已处理的MD5
				rows, _ := existingFile.GetRows("Sheet1")
				for rowIdx, row := range rows {
					if rowIdx < 2 { // 跳过前两行（问题和标题）
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

	// 创建新文件（如果不存在或问题不同）
	if f == nil {
		f = excelize.NewFile()
		// 设置首行为问题
		f.SetCellValue("Sheet1", "A1", input)
		// 设置标题行
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
		filePath := filePath // 创建局部变量副本
		i := i

		if progress != nil {
			progress <- tui.ProgressMsg{Index: i, Total: totalFiles, Filename: fpath.Base(filePath), Status: "processing"}
		}

		pool.Submit(func() {
			defer wg.Done()

			if progress == nil {
				console.Colorful(fmt.Sprintf("\n🔄 [%d/%d] 正在处理: %s", i+1, len(pendingFiles), fpath.Base(filePath)), constant.Blue)
			}

			answer, err := sendRequestWithFile(input, filePath)
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

			// 写入Excel数据
			fileName := file.RemoveFileExtension(fpath.Base(filePath))
			md5, _ := file.CalculateMD5(filePath)

			// 写入数据
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
			// 实时保存进度
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

func runDIYQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
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

	// 读取Excel文件
	if progress == nil {
		fmt.Println("\n📂 正在打开 process.xlsx...")
	}
	excelFile, err := excelize.OpenFile("process.xlsx")
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer excelFile.Close()

	// 获取所有行数据
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

	totalRows := len(rows) - 1 // skip header

	defer excelFile.Save()
	for i, row := range rows {
		if i == 0 {
			continue // 跳过标题行
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

			answer, err := sendRequestWithFile(input, filePath)
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

			// 写入Excel数据
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

func printQuestion(question string, colors ...string) {
	color := constant.Blue
	if len(colors) > 0 {
		color = colors[0]
	}
	fmt.Println(color + question + constant.Reset)
}

// 显示动态 Loading 动画
func showLoading(done chan struct{}) {
	frames := []string{"-", "\\", "|", "/"}
	i := 0

	for {
		select {
		case <-done:
			return
		default:
			// fmt.Printf("\r"+constant.Yellow+"正在处理中 %s"+constant.Reset, frames[i%len(frames)])
			fmt.Printf("\r"+constant.Yellow+"%s"+constant.Reset, frames[i%len(frames)])
			i++
			time.Sleep(80 * time.Millisecond) // 控制旋转速度
		}
	}
}

func runWorkflowQueryRule(poolSize int, progress chan<- tui.ProgressMsg) error {
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	if progress == nil {
		console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), constant.Green)
	}

	start := time.Now()

	// 读取Excel文件
	if progress == nil {
		fmt.Println("\n📂 正在打开 workflow.xlsx...")
	}
	file, err := excelize.OpenFile("workflow.xlsx")
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 获取所有行数据
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

	head := rows[0] // 第一行作为表头

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	var errCount int
	totalRows := len(rows) - 1

	// 遍历每一行
	for i, row := range rows {
		if i == 0 {
			continue // 跳过标题行
		}

		// 问题在第一列
		if len(row) == 0 {
			continue
		}
		question := row[0]

		// 如果第二列有值，说明已经处理过
		if len(row) >= 2 && row[1] != "" {
			if progress != nil {
				progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "skip"}
			}
			continue
		}

		// 深拷贝一份，避免闭包问题
		iCopy := i
		rowCopy := append([]string{}, row...)

		if progress == nil {
			printQuestion(question)
		} else {
			progress <- tui.ProgressMsg{Index: i, Total: totalRows, Filename: question, Status: "processing"}
		}

		wg.Add(1)
		_ = pool.Submit(func() {
			defer wg.Done()

			// 构造参数 map（从第3列开始）
			argsM := make(map[string]string)
			for col := 2; col < len(head) && col < len(rowCopy); col++ {
				argsM[head[col]] = rowCopy[col]
			}

			// 发送请求并获取响应
			client := bailian.NewClientWithAppIDAPIKey(appId, apiKey)
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

			// 将结果写入到 Excel 的第二列
			mu.Lock()
			file.SetCellValue("Sheet1", fmt.Sprintf("B%d", iCopy+1), response.Output.Text)
			mu.Unlock()

			if progress != nil {
				progress <- tui.ProgressMsg{Index: iCopy, Total: totalRows, Filename: question, Status: "done"}
			}
		})
	}

	// 等待所有并发请求完成
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
