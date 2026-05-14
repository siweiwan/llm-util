package bailian

import (
	"context"
	"fmt"
	"io"
	"testing"
)

const (
	apiKey = "your-api-key"
)

func TestCreateChatCompletion(t *testing.T) {
	client := NewClientWithAppIDAPIKey("1f03bff2a0f74eae9e1b553f980cfdd6", apiKey)
	response, err := client.CreateChatCompletion(context.TODO(), ChatCompletionRequest{
		Input: &RequestInput{
			Prompt: "你是谁？",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(response)
}

func TestCreateChatCompletionStream(t *testing.T) {
	client := NewClientWithAppIDAPIKey("1f03bff2a0f74eae9e1b553f980cfdd6", apiKey)
	streamMsg, err := client.CreateChatCompletionStream(context.TODO(), ChatCompletionRequest{
		Input: &RequestInput{
			Prompt: "你是谁？",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}

	defer streamMsg.Close()

	for {
		recv, err := streamMsg.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Printf("Stream chat error: %v\n", err)
			return
		}
		fmt.Printf("Stream chat: %v\n", recv.Output.Text)

	}
}

func TestCreateChatCompletionStreamBizParams(t *testing.T) {

	client := NewClientWithAppIDAPIKey("096f858b44614d1b9a676177276c2899", apiKey)
	streamMsg, err := client.CreateChatCompletionStream(context.TODO(), ChatCompletionRequest{
		Input: &RequestInput{
			Prompt: " ",
			BizParams: RequestInputBizParams{
				UserPromptParams: map[string]interface{}{
					"project_name": "正负电子对撞机国家实验室",
				},
			},
		},
		Parameters: &RequestParameters{
			IncrementalOutput: true, // 由于要自行拼接响应，选择增量输出模式
			HasThoughts:       true, // 是否展示思考过程
		},
	})
	if err != nil {
		t.Error(err)
		return
	}

	defer streamMsg.Close()

	for {
		recv, err := streamMsg.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Printf("Stream chat error: %v\n", err)
			return
		}
		if recv.Output == nil {
			return
		}

		if len(recv.Output.Thoughts) > 0 {
			if recv.Output.Thoughts[0].Thought != "" {
				fmt.Printf("[thought]%s\n", recv.Output.Thoughts[0].Thought)
			}
			if recv.Output.Thoughts[0].ReasoningContent != "" {
				fmt.Printf("[reasoning]%s\n", recv.Output.Thoughts[0].ReasoningContent)
			}
			if recv.Output.Text != "" {
				fmt.Printf("[text]%s\n", recv.Output.Text)
			}
		}

	}
}
