package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llm-util/ai/app/impl/bailian"
	"llm-util/file/qwen"
	"llm-util/util"
	"llm-util/util/console"
	"llm-util/util/file"
	"net/http"
	"os"
	fpath "path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"

	"github.com/xuri/excelize/v2"
)

var apiKey = ""
var appId = ""

// var appId = "1f03bff2a0f74eae9e1b553f980cfdd6"

var conversationHistory []Message

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ANSI 颜色代码
const (
	Reset     = "\033[0m"
	Blue      = "\033[34m"
	Green     = "\033[32m"
	Red       = "\033[31m"
	Yellow    = "\033[33m"
	ClearLine = "\033[2K\r" // 清除当前行并回到行首
)

func main() {
	// 获取API Key
	if apiKey == "" {
		fmt.Print("请输入API Key(请妥善保管您的apiKey,请勿泄露给他人): ")
		fmt.Scanln(&apiKey)
	}

	if appId == "" {
		fmt.Print(`使用调用文件上传请设置支持文件上传的AppId。
请输入AppId: `)
		fmt.Scanln(&appId)
	}

	for {
		fmt.Println("\n" + strings.Repeat("=", 80))
		console.Colorful("                           主菜单", Yellow)
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("  【1】开始/继续对话 (自由模式)")
		fmt.Println("  【2】新对话 (自由模式)")
		fmt.Println("  【3】退出程序")
		fmt.Println("  【4】规则模式")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Print("请输入选项 (1-4): ")

		var choice int
		_, err := fmt.Scanln(&choice)
		if err != nil {
			fmt.Println("❌ 无效输入，请重新选择")
			continue
		}

		switch choice {
		case 1:
			startConversation()
		case 2:
			resetConversation()
			console.Colorful("\n✅ 已开启新对话", Green)
			startConversation()
		case 3:
			console.Colorful("\n👋 再见！感谢使用！", Yellow)
			return
		case 4:
			runRuleMode()
		default:
			fmt.Println("❌ 无效选项，请重新选择")
		}
	}
}

func startConversation() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n" + strings.Repeat("-", 80))
		console.Colorful("💬 请输入您的问题", Blue)
		fmt.Println("   (输入 'exit' 返回主菜单)")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.ToLower(input) == "exit" {
			console.Colorful("\n⬅️  返回主菜单", Yellow)
			return
		}

		// 添加用户消息到历史
		conversationHistory = append(conversationHistory, Message{
			Role:    "user",
			Content: input,
		})

		// 发送请求并获取响应
		response, err := sendRequest(input)
		if err != nil {
			console.Colorful(fmt.Sprintf("\n❌ 请求失败: %v", err), Red)
			continue
		}

		// 添加助手响应到历史
		conversationHistory = append(conversationHistory, Message{
			Role:    "assistant",
			Content: response,
		})

		printResp(response)
	}
}

func resetConversation() {
	conversationHistory = []Message{}
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

	// 创建一个通道用于终止 loading 动画
	// done := make(chan struct{})
	// go showLoading(done)

	resp, err := client.Do(req)
	// close(done)
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

func runRuleMode() {
	// 	fmt.Println(`1. 规则1【案例查询】
	// - 请在A列写入提问，B列为AI回复。（第一行请留给标题）
	// - C1为二次提问，为空采取默认提问:
	// 请根据提供内容，对“服务国家和地区重大战略需求、学科优势和特色、学科点的不可替代性”三个主题，每个主题总结10个切入点，并且对每个切入点进行举例。以表格输出给我，字段包括：主题、切入点、案例
	// - C2为AI回复。`)

	console.Colorful("1. 规则1【案例查询】", Red)
	fmt.Println(`- 请在当前目录 data.xlsx A列写入提问，B列为AI回复。（第一行请留给标题）`)

	console.Colorful("2. 规则2【PDF批量提问】", Red)
	fmt.Println(`- 请在当前目录下新建 pdfs 文件夹并放至需要提问的 PDF 文件。
- 会自动将回答输出至当前目录下的 {提问前20个字}.xlsx 文件。
- B列的 md5 值用于跳过已处理过的文件。`)

	console.Colorful("3. 规则3【DIY提问】", Red)
	fmt.Println(`(用于处理 n×m 的提问规模，n 为问题数，m 为文件数)
- 请在当前目录下新建 files 文件夹并放至需要提问的文件。
- process.xlsx 为处理模板，首行标题不处理
eg:
  A              B             C
1 [ 问题  ]  [   文件   ]   [  回答   ]
2 [ 问题1 ]  [ 1.pdf   ]   [ answer1 ]
3 [ 问题2 ]  [ 2.word  ]   [ answer2 ]
4 [ 问题3 ]  [ 3.excel ]   [ answer3 ]`)

	console.Colorful("4. 规则4【工作流调用】", Red)
	fmt.Println(`- workflow.xlsx 为处理模板，首行标题不处理
- 第一列为问题，第二列输出回答，表头决定了参数
eg:
  A              B             C
1 [ question ]  [ answer ]  [ url ]    [ name ]
2 [ 请给出回答 ]  [ 歪比巴卜 ]  [ .com ]  [ 名字 ]`)

	fmt.Println("请选择规则模式：")

	var ruleChoice int
	fmt.Print("> ")
	_, err := fmt.Scanln(&ruleChoice)
	if err != nil {
		fmt.Println("无效输入，请重新选择")
		return
	}

	switch ruleChoice {
	case 1:
		console.Colorful("\n▶️  开始执行：规则1 - 案例查询", Green)
		runCaseQueryRule()
	case 2:
		console.Colorful("\n▶️  开始执行：规则2 - PDF批量提问", Green)
		runPdfBatchQuery()
	case 3:
		console.Colorful("\n▶️  开始执行：规则3 - DIY提问", Green)
		runDIYQueryRule()
	case 4:
		console.Colorful("\n▶️  开始执行：规则4 - 工作流调用", Green)
		runWorkflowQueryRule()
	default:
		fmt.Println("❌ 无效选项，请重新选择")
	}
}

func runCaseQueryRule() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n⚙️  请输入并发规模 (建议: 10-50, 过大可能导致请求频率限制):")
	fmt.Print("> ")
	poolSizeStr, _ := reader.ReadString('\n')
	poolSize, _ := strconv.Atoi(strings.TrimSpace(poolSizeStr))
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), Green)

	start := time.Now()

	// 读取Excel文件
	fmt.Println("\n📂 正在打开 data.xlsx...")
	file, err := excelize.OpenFile("data.xlsx")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 打开文件失败: %v", err), Red)
		return
	}
	defer file.Close()

	// 获取所有行数据
	rows, err := file.GetRows("Sheet1")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 读取行数据失败: %v", err), Red)
		return
	}
	console.Colorful(fmt.Sprintf("✅ 成功读取 %d 行数据", len(rows)), Green)

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	// 创建ants协程池（根据需要调整池大小）
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	var (
		cacheChat string
		errCount  int
	)

	// 用于并发发送请求
	for i, row := range rows {
		if i == 0 {
			continue // 跳过标题行
		}

		// university := row[0]
		// subject := row[1]
		// angle := row[2]

		question := row[0]
		if len(row) >= 2 {
			if row[1] != "" {
				// 跳过已处理过的文件
				continue
			}
		}

		// output := fmt.Sprintf("搜索一下。%s %s学科 在切入点 %s 有哪些具体案例。", university, subject, angle)
		// 搜索一下。江西师范大学 外国语言文学学科 在切入点 提升国际传播效能 有哪些具体案例。
		printQuestion(question)

		wg.Add(1)
		pool.Submit(func() {
			defer wg.Done()
			i := i
			question := question

			// 发送请求并获取响应
			response, err := sendRequest(question)
			if err != nil {
				fmt.Printf("请求失败: %v\n", err)
				errCount++
				return
			}

			// 将结果写入到 Excel 的第四列
			mu.Lock()
			// 添加助手响应到历史
			cacheChat += response
			file.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+1), response)
			mu.Unlock()
		})
	}

	// 等待所有并发请求完成
	wg.Wait()

	// 保存修改后的 Excel 文件
	fmt.Println("\n💾 正在保存 Excel 文件...")
	if err := file.Save(); err != nil {
		console.Colorful(fmt.Sprintf("❌ 保存文件失败: %v", err), Red)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	console.Colorful(fmt.Sprintf("✅ 规则模式【案例查询】处理完毕！耗时: %v", time.Since(start)), Yellow)
	fmt.Println(strings.Repeat("=", 80))
	if errCount == 0 {
		console.Colorful("🎉🎉🎉 所有请求成功完成！", Green)
	} else {
		console.Colorful(fmt.Sprintf("⚠️  请求失败数量: %d", errCount), Red)
	}
}

// 示例：规则2 单问题批量提问
func runPdfBatchQuery() {
	fmt.Println("\n🔍 文件检索中...")
	// 1.检索当前文件夹下有多少 PDF 文件
	dir, err := os.Getwd()
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 获取当前目录失败: %v", err), Red)
		return
	}
	files, err := file.GetFiles(dir+"/pdfs", "pdf")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 读取文件失败: %v", err), Red)
		return
	}
	console.Colorful(fmt.Sprintf("✅ 在 pdfs 目录下检索到 %d 个 PDF 文件", len(files)), Green)

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n⚙️  请输入并发规模 (过大可能导致请求频率限制):")
	fmt.Print("> ")
	poolSizeStr, _ := reader.ReadString('\n')
	poolSize, _ := strconv.Atoi(strings.TrimSpace(poolSizeStr))
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), Green)

	fmt.Println("\n💬 请输入您的问题:")
	fmt.Print("> ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

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
				console.Colorful("📋 检测到未完成的进度文件，将继续处理...", Yellow)

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
				console.Colorful("⚠️  检测到同名文件但问题不同，将创建新文件", Yellow)
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
			console.Colorful(fmt.Sprintf("❌ 保存Excel文件失败: %v", err), Red)
		} else {
			console.Colorful(fmt.Sprintf("\n✅ 所有结果已保存至: %s", filename), Green)
		}
		if err := f.Close(); err != nil {
			console.Colorful(fmt.Sprintf("❌ 关闭Excel文件失败: %v", err), Red)
		}
	}()

	// 设置首行为问题
	f.SetCellValue("Sheet1", "A1", input)

	// 设置第二行标题
	headers := []string{"文件名", "MD5", "回答内容"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		f.SetCellValue("Sheet1", cell, header)
	}

	var pendingFiles []string
	for _, filePath := range files {
		md5, err := file.CalculateMD5(filePath)
		if err == nil && processedMD5[md5] {
			console.Colorful(fmt.Sprintf("⏭️  跳过已处理文件: %s", fpath.Base(filePath)), Yellow)
			continue
		}
		pendingFiles = append(pendingFiles, filePath)
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
	console.Colorful(fmt.Sprintf("📊 需要新处理的文件数量/总文件数量: %d/%d", len(pendingFiles), len(files)), Blue)
	fmt.Println(strings.Repeat("-", 80))

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 创建ants协程池（根据需要调整池大小）
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	for i, filePath := range pendingFiles {
		wg.Add(1)
		filePath := filePath // 创建局部变量副本
		i := i

		pool.Submit(func() {
			defer wg.Done()

			console.Colorful(fmt.Sprintf("\n🔄 [%d/%d] 正在处理: %s", i+1, len(pendingFiles), fpath.Base(filePath)), Blue)

			answer, err := sendRequestWithFile(input, filePath)
			if err != nil {
				console.Colorful(fmt.Sprintf("❌ 文件[%s] 请求失败: %v", fpath.Base(filePath), err), Red)
				return
			}
			console.Colorful(fmt.Sprintf("✅ [%d/%d] %s 处理完成", i+1, len(pendingFiles), fpath.Base(filePath)), Green)
			fmt.Printf(Green+"\n📄 %s 回答内容:\n%s\n"+Reset, fpath.Base(filePath), answer)

			// 写入Excel数据
			fileName := file.RemoveFileExtension(fpath.Base(filePath))
			md5, _ := file.CalculateMD5(filePath)

			// 写入数据
			data := []string{fileName, md5, answer}

			for col, value := range data {
				cell, _ := excelize.CoordinatesToCellName(col+1, currentRow)
				if err := f.SetCellValue("Sheet1", cell, value); err != nil {
					console.Colorful(fmt.Sprintf("⚠️  写入数据到%s失败: %v", cell, err), Yellow)
				}
			}

			mu.Lock()
			currentRow++
			// 实时保存进度
			if err := f.SaveAs(filename); err != nil {
				console.Colorful(fmt.Sprintf("⚠️  临时保存失败: %v", err), Yellow)
			}
			mu.Unlock()
		})
	}
	wg.Wait()
}

func runDIYQueryRule() {
	fmt.Println("\n🔍 文件检索中...")
	// 1.检索当前文件夹下有多少文件
	dir, err := os.Getwd()
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 获取当前目录失败: %v", err), Red)
		return
	}
	files, err := file.GetFiles(dir+"/files", "")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 读取文件失败: %v", err), Red)
		return
	}
	console.Colorful(fmt.Sprintf("✅ 在 files 目录下检索到 %d 个文件", len(files)), Green)

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n⚙️  请输入并发规模 (过大可能导致请求频率限制):")
	fmt.Print("> ")
	poolSizeStr, _ := reader.ReadString('\n')
	poolSize, _ := strconv.Atoi(strings.TrimSpace(poolSizeStr))
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), Green)

	// 读取Excel文件
	fmt.Println("\n📂 正在打开 process.xlsx...")
	file, err := excelize.OpenFile("process.xlsx")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 打开文件失败: %v", err), Red)
		return
	}
	defer file.Close()

	// 获取所有行数据
	rows, err := file.GetRows("Sheet1")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 读取行数据失败: %v", err), Red)
		return
	}
	console.Colorful(fmt.Sprintf("✅ 成功读取 %d 行数据", len(rows)), Green)

	var wg sync.WaitGroup
	var mu sync.Mutex
	// 创建ants协程池（根据需要调整池大小）
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	defer file.Save()
	for i, row := range rows {
		if i == 0 {
			continue // 跳过标题行
		}

		if len(row) < 2 {
			continue
		} else if len(row) > 2 {
			if row[2] != "" {
				continue
			}
		} else {
			if row[0] == "" || row[1] == "" {
				continue
			}
		}

		question := row[0]
		fileName := row[1]

		wg.Add(1)
		pool.Submit(func() {
			i := i
			input := question
			filePath := dir + "/files/" + fileName

			defer wg.Done()

			console.Colorful(fmt.Sprintf("\n🔄 正在处理 %s", fileName), Blue)
			console.Colorful(fmt.Sprintf("   问题: %s", input), Blue)

			answer, err := sendRequestWithFile(input, filePath)
			if err != nil {
				console.Colorful(fmt.Sprintf("❌ 文件[%s] 请求失败: %v", fileName, err), Red)
				return
			}
			console.Colorful(fmt.Sprintf("✅ %s 处理完成", fileName), Green)
			fmt.Printf(Green+"\n📄 %s 回答内容:\n%s\n"+Reset, fileName, answer)

			// 写入Excel数据
			mu.Lock()
			file.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+1), answer)
			file.Save()
			mu.Unlock()
		})
	}
}

func printResp(resp string, colors ...string) {
	color := Green
	if len(colors) > 0 {
		color = colors[0]
	}
	fmt.Println("\n🧑‍💻 " + color + "助手回复:" + Reset)
	fmt.Println(color + resp + Reset)
}

func printQuestion(question string, colors ...string) {
	color := Blue
	if len(colors) > 0 {
		color = colors[0]
	}
	fmt.Println(color + question + Reset)
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
			// fmt.Printf("\r"+Yellow+"正在处理中 %s"+Reset, frames[i%len(frames)])
			fmt.Printf("\r"+Yellow+"%s"+Reset, frames[i%len(frames)])
			i++
			time.Sleep(80 * time.Millisecond) // 控制旋转速度
		}
	}
}

func runWorkflowQueryRule() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n⚙️  请输入并发规模 (过大可能导致请求频率限制):")
	fmt.Print("> ")
	poolSizeStr, _ := reader.ReadString('\n')
	poolSize, _ := strconv.Atoi(strings.TrimSpace(poolSizeStr))
	if poolSize <= 0 {
		poolSize = 10
	} else if poolSize > 200 {
		poolSize = 200
	}
	console.Colorful(fmt.Sprintf("✅ 并发规模已设置为: %d", poolSize), Green)

	start := time.Now()

	// 读取Excel文件
	fmt.Println("\n📂 正在打开 workflow.xlsx...")
	file, err := excelize.OpenFile("workflow.xlsx")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 打开文件失败: %v", err), Red)
		return
	}
	defer file.Close()

	// 获取所有行数据
	rows, err := file.GetRows("Sheet1")
	if err != nil {
		console.Colorful(fmt.Sprintf("❌ 读取行数据失败: %v", err), Red)
		return
	}
	if len(rows) < 2 {
		console.Colorful("❌ Excel中没有数据", Red)
		return
	}
	console.Colorful(fmt.Sprintf("✅ 成功读取 %d 行数据", len(rows)), Green)

	head := rows[0] // 第一行作为表头

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	pool, _ := ants.NewPool(poolSize)
	defer pool.Release()

	var errCount int

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
			continue
		}

		// 深拷贝一份，避免闭包问题
		iCopy := i
		rowCopy := append([]string{}, row...)

		wg.Add(1)
		_ = pool.Submit(func() {
			defer wg.Done()

			// 构造参数 map（从第3列开始）
			argsM := make(map[string]string)
			for col := 2; col < len(head) && col < len(rowCopy); col++ {
				argsM[head[col]] = rowCopy[col]
			}

			printQuestion(question)

			// 发送请求并获取响应
			client := bailian.NewClientWithAppIDAPIKey(appId, apiKey)
			response, err := client.CreateChatCompletion(context.TODO(), bailian.ChatCompletionRequest{
				Input: &bailian.RequestInput{
					Prompt:    question,
					BizParams: argsM,
				},
			})
			if err != nil {
				console.Colorful(fmt.Sprintf("❌ 请求失败: %v", err), Red)
				mu.Lock()
				errCount++
				mu.Unlock()
				return
			}

			console.Colorful(fmt.Sprintf("✅ 问题 [%d] 处理完成", iCopy+1), Green)

			// 将结果写入到 Excel 的第二列
			mu.Lock()
			file.SetCellValue("Sheet1", fmt.Sprintf("B%d", iCopy+1), response.Output.Text)
			mu.Unlock()
		})
	}

	// 等待所有并发请求完成
	wg.Wait()

	// 保存修改后的 Excel 文件
	fmt.Println("\n💾 正在保存 Excel 文件...")
	if err := file.Save(); err != nil {
		console.Colorful(fmt.Sprintf("❌ 保存文件失败: %v", err), Red)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	console.Colorful(fmt.Sprintf("✅ 规则模式【工作流】处理完毕！耗时: %v", time.Since(start)), Yellow)
	fmt.Println(strings.Repeat("=", 80))
	if errCount == 0 {
		console.Colorful("🎉🎉🎉 所有请求成功完成！", Green)
	} else {
		console.Colorful(fmt.Sprintf("⚠️  请求失败数量: %d", errCount), Red)
	}
}
