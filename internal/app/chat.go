package app

import (
	"context"
	"fmt"
	"llm-util/ai/app/impl/bailian"
	"llm-util/conf"
	uploadfile "llm-util/file"
	"log/slog"
	"time"
)

// newClient 基于当前配置创建百炼 API 客户端
func (a *App) newClient() *bailian.Client {
	cfg := bailian.NewClientConfig(a.AppId, a.APIKey,
		bailian.WithWorkspace(conf.WORKSPACE_ID),
	)
	return bailian.NewClientWithConfig(cfg)
}

// toMessages 将内部对话历史转换为百炼请求消息
func (a *App) toMessages() []bailian.RequestMessage {
	if len(a.History) == 0 {
		return nil
	}
	msgs := make([]bailian.RequestMessage, len(a.History))
	for i, m := range a.History {
		msgs[i] = bailian.RequestMessage{
			Content: m.Content,
			Role:    m.Role,
		}
	}
	return msgs
}

// SendRequest 发送纯文本提问
func (a *App) SendRequest(prompt string) (string, error) {
	slog.Info("SendRequest", "prompt", prompt)
	resp, err := a.newClient().CreateChatCompletion(context.TODO(), bailian.ChatCompletionRequest{
		Input: &bailian.RequestInput{
			Prompt:   prompt,
			Messages: a.toMessages(),
		},
	})
	if err != nil {
		slog.Error("SendRequest failed", "prompt", prompt, "err", err)
		return "", err
	}
	slog.Info("SendRequest done", "response_len", len(resp.Output.Text))
	return resp.Output.Text, nil
}

const maxUploadRetries = 2 // 1 次初始 + 1 次重试

// SendRequestWithFile 上传文件后带文件提问，上传失败时自动重试 1 次
func (a *App) SendRequestWithFile(prompt, filePath string) (string, error) {
	slog.Info("SendRequestWithFile", "prompt", prompt, "file", filePath)

	var fileId string
	var err error
	for attempt := 1; attempt <= maxUploadRetries; attempt++ {
		fileId, err = uploadfile.UploadFile(filePath)
		if err == nil {
			break
		}
		slog.Error("SendRequestWithFile upload failed", "file", filePath, "attempt", attempt, "err", err)
		if attempt < maxUploadRetries {
			slog.Info("重试上传", "file", filePath, "next_attempt", attempt+1)
			time.Sleep(3 * time.Second) // 重试前等待
		}
	}
	if err != nil {
		return "", fmt.Errorf("上传文件失败(已重试%d次): %w", maxUploadRetries-1, err)
	}
	slog.Info("文件上传成功", "fileId", fileId)

	resp, err := a.newClient().CreateChatCompletion(context.TODO(), bailian.ChatCompletionRequest{
		Input: &bailian.RequestInput{
			Prompt:   prompt,
			Messages: a.toMessages(),
		},
		Parameters: &bailian.RequestParameters{
			RagOptions: &bailian.RequestInputRagOptions{
				SessionFileIDs: []string{fileId},
			},
		},
	})
	if err != nil {
		slog.Error("SendRequestWithFile chat failed", "fileId", fileId, "err", err)
		return "", err
	}
	slog.Info("SendRequestWithFile done", "fileId", fileId, "response_len", len(resp.Output.Text))
	return resp.Output.Text, nil
}
