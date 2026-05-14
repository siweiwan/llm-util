package qwen

import (
	"fmt"
	"llm-util/conf"
	"testing"
)

func TestSend(t *testing.T) {
	req := ApplyFileUploadLeaseRequest{
		CategoryId:   "default",
		WorkspaceId:  conf.WORKSPACE_ID,
		FileName:     "10110中北大学0701数学申博.pdf",
		Md5:          "8a818f8b5c0469be25f318353ff559ba",
		SizeInBytes:  "7575890",
		CategoryType: CategoryType_SESSION_FILE,
	}
	resp, err := req.Send()
	if err != nil {
		return
	}
	fmt.Println(resp)
}

func TestCreateUploadRequest(t *testing.T) {

}
