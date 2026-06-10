package conf

// Config 应用配置，所有可配置项集中管理
type Config struct {
	APIKey      string
	AppID       string
	WorkspaceID string
	PoolSize    int
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		WorkspaceID: "llm-shwq55idtv5plnag",
		PoolSize:    4,
	}
}

// WORKSPACE_ID 兼容文件上传模块直接引用，main 启动时从 Config 同步
var WORKSPACE_ID = "llm-shwq55idtv5plnag"
