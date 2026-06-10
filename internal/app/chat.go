package app

import (
	"context"
	"fmt"
	"llm-util/ai/app/impl/bailian"
	"llm-util/conf"
	uploadfile "llm-util/file"
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
	resp, err := a.newClient().CreateChatCompletion(context.TODO(), bailian.ChatCompletionRequest{
		Input: &bailian.RequestInput{
			Prompt:   prompt,
			Messages: a.toMessages(),
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Output.Text, nil
}

// SendRequestWithFile 上传文件后带文件提问
func (a *App) SendRequestWithFile(prompt, filePath string) (string, error) {
	fileId, err := uploadfile.UploadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %w", err)
	}

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
		return "", err
	}
	return resp.Output.Text, nil
}
