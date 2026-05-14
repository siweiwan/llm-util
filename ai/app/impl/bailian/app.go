package bailian

import (
	"context"
	"llm-util/ai/app/abstract/schema"
	"fmt"
)

type AppConfig struct {
	// default:https://dashscope.aliyuncs.com/api/v1/apps
	BaseURL string `json:"baseUrl"`
	ApiKey  string `json:"apiKey"`
	// https://help.aliyun.com/zh/model-studio/developer-reference/obtain-api-key-app-id-and-workspace-id?spm=a2c4g.11186623.0.0.54fc4823QedxG4
	// 百炼应用ID
	AppID string `json:"appId"`
	// 业务空间标识
	WorkSpaceID string `json:"workspaceId"`
}

type App struct {
	cli *Client
}

func NewApp(ctx context.Context, cfg *AppConfig) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ai app config is nil")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://dashscope.aliyuncs.com/api/v1/apps"
	}
	if cfg.ApiKey == "" {
		return nil, fmt.Errorf("ai app apiKey is empty")
	}
	if cfg.AppID == "" {
		return nil, fmt.Errorf("ai app appId is empty")
	}

	return &App{}, nil
}

func (a *App) Generate(ctx context.Context, in []*schema.Message) (*schema.Message, error) {

	return nil, nil
}

func (a *App) Stream() (*schema.Message, error) {

	return nil, nil
}
