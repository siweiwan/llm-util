package qwen

import (
	"fmt"
	"testing"
)

func TestUploadFile(t *testing.T) {
	filePath := "D:\\goproject\\src\\llm\\files\\10110中北大学0701数学申博.pdf"
	_, err := UploadFile(filePath)
	fmt.Println(err)
}
