package app

import (
	"context"
	"llm-util/ai/app/impl/bailian"
	uploadfile "llm-util/file"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func setupApp(t *testing.T) *App {
	t.Helper()
	_ = godotenv.Load("../../.env")
	apiKey := os.Getenv("LLM_API_KEY")
	appId := os.Getenv("LLM_APP_ID")
	if apiKey == "" || appId == "" {
		t.Skip("跳过: 未设置 LLM_API_KEY 或 LLM_APP_ID")
	}
	return New(apiKey, appId)
}

// TestSendRequest 纯文本提问，验证 bailian.Client 调用链路
func TestSendRequest(t *testing.T) {
	a := setupApp(t)

	answer, err := a.SendRequest("你好，请用一句话介绍一下你自己")
	if err != nil {
		t.Fatalf("SendRequest 失败: %v", err)
	}
	if answer == "" {
		t.Fatal("返回内容为空")
	}
	t.Logf("回复: %s", answer)
}

// TestSendRequestWithFile 上传文件 + 查询状态 + 带文件提问，端到端验证
func TestSendRequestWithFile(t *testing.T) {
	a := setupApp(t)

	testFile := "../../files/1.txt"
	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("跳过: 测试文件不存在 %s", testFile)
	}

	// Step 1: 上传文件
	t.Log("Step 1: 上传文件...")
	fileId, err := uploadfile.UploadFile(testFile)
	if err != nil {
		t.Fatalf("上传文件失败: %v", err)
	}
	t.Logf("文件上传完成, fileId: %s", fileId)

	// Step 2: 查询文件状态，确认就绪
	t.Log("Step 2: 查询文件状态...")
	status, err := uploadfile.DescribeFileStatus(fileId)
	if err != nil {
		t.Fatalf("查询文件状态失败: %v", err)
	}
	t.Logf("文件状态: %s", status)
	if status != "FILE_IS_READY" && status != "PARSE_SUCCESS" {
		t.Fatalf("文件未就绪，当前状态: %s", status)
	}

	// Step 3: 带文件提问
	t.Log("Step 3: 带文件提问...")
	resp, err := a.newClient().CreateChatCompletion(context.TODO(), bailian.ChatCompletionRequest{
		Input: &bailian.RequestInput{
			Prompt: "请阅读这个文件并告诉我文件里写了什么",
		},
		Parameters: &bailian.RequestParameters{
			RagOptions: &bailian.RequestInputRagOptions{
				SessionFileIDs: []string{fileId},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion 失败: %v", err)
	}
	t.Logf("回复: %s", resp.Output.Text)
}
