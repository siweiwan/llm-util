package app

import (
	"fmt"
	"llm-util/conf"
	"os"
	"strings"
)

func EnsureEnvFile() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		f, err := os.Create(".env")
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString("LLM_API_KEY=\nLLM_APP_ID=\nWORKSPACE_ID=llm-shwq55idtv5plnag\nPOOL_SIZE=10\nALIBABA_CLOUD_ACCESS_KEY_ID=\nALIBABA_CLOUD_ACCESS_KEY_SECRET=\n")
	}
}

func SaveEnvFile(cfg *conf.Config) error {
	data, err := os.ReadFile(".env")
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	updatedKey, updatedId, updatedWs, updatedPool := false, false, false, false
	updatedAkId, updatedAkSecret := false, false
	for i, line := range lines {
		if strings.HasPrefix(line, "LLM_API_KEY=") {
			lines[i] = "LLM_API_KEY=" + cfg.APIKey
			updatedKey = true
		}
		if strings.HasPrefix(line, "LLM_APP_ID=") {
			lines[i] = "LLM_APP_ID=" + cfg.AppID
			updatedId = true
		}
		if strings.HasPrefix(line, "WORKSPACE_ID=") {
			lines[i] = "WORKSPACE_ID=" + cfg.WorkspaceID
			updatedWs = true
		}
		if strings.HasPrefix(line, "POOL_SIZE=") {
			lines[i] = fmt.Sprintf("POOL_SIZE=%d", cfg.PoolSize)
			updatedPool = true
		}
		if strings.HasPrefix(line, "ALIBABA_CLOUD_ACCESS_KEY_ID=") {
			lines[i] = "ALIBABA_CLOUD_ACCESS_KEY_ID=" + cfg.AccessKeyId
			updatedAkId = true
		}
		if strings.HasPrefix(line, "ALIBABA_CLOUD_ACCESS_KEY_SECRET=") {
			lines[i] = "ALIBABA_CLOUD_ACCESS_KEY_SECRET=" + cfg.AccessKeySecret
			updatedAkSecret = true
		}
	}
	if !updatedKey {
		lines = append(lines, "LLM_API_KEY="+cfg.APIKey)
	}
	if !updatedId {
		lines = append(lines, "LLM_APP_ID="+cfg.AppID)
	}
	if !updatedWs {
		lines = append(lines, "WORKSPACE_ID="+cfg.WorkspaceID)
	}
	if !updatedPool {
		lines = append(lines, fmt.Sprintf("POOL_SIZE=%d", cfg.PoolSize))
	}
	if !updatedAkId {
		lines = append(lines, "ALIBABA_CLOUD_ACCESS_KEY_ID="+cfg.AccessKeyId)
	}
	if !updatedAkSecret {
		lines = append(lines, "ALIBABA_CLOUD_ACCESS_KEY_SECRET="+cfg.AccessKeySecret)
	}
	return os.WriteFile(".env", []byte(strings.Join(lines, "\n")), 0644)
}
