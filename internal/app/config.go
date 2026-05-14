package app

import (
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
		f.WriteString("LLM_API_KEY=\nLLM_APP_ID=\nALIBABA_CLOUD_ACCESS_KEY_ID=\nALIBABA_CLOUD_ACCESS_KEY_SECRET=\n")
	}
}

func SaveEnvFile(apiKey, appId string) error {
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
