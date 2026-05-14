package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llm-util/file/qwen"
	"net/http"
)

func (a *App) SendRequest(prompt string) (string, error) {
	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/apps/%s/completion", a.AppId)

	requestBody := map[string]interface{}{
		"input": map[string]interface{}{
			"prompt":   prompt,
			"messages": a.History,
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

	req.Header.Set("Authorization", "Bearer "+a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	done := make(chan struct{})
	go ShowLoading(done)

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

func (a *App) SendRequestWithFile(prompt, filePath string) (string, error) {
	addFileResponse, err := qwen.UploadFile(filePath)
	if err != nil {
		fmt.Printf("上传文件失败: %v\n", err)
		return "", err
	}
	sessionFileId := *addFileResponse.Body.Data.FileId

	url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/apps/%s/completion", a.AppId)

	requestBody := map[string]interface{}{
		"input": map[string]interface{}{
			"prompt":   prompt,
			"messages": a.History,
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

	req.Header.Set("Authorization", "Bearer "+a.APIKey)
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
