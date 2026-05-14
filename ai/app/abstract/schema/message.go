/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package schema

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/slongfield/pyfmt"
)

type RoleType string

const (
	System    RoleType = "system"
	User      RoleType = "user"
	Assistant RoleType = "assistant"
)

// FormatType used by MessageTemplate.Format
type FormatType uint8

const (
	// FString Supported by pyfmt(github.com/slongfield/pyfmt), which is an implementation of https://peps.python.org/pep-3101/.
	FString FormatType = 0
	// GoTemplate https://pkg.go.dev/text/template.
	GoTemplate FormatType = 1
	// Jinja2 Supported by gonja(github.com/nikolalohinski/gonja), which is a implementation of https://jinja.palletsprojects.com/en/3.1.x/templates/.
	Jinja2 FormatType = 2
)

type Message struct {
	Role    RoleType `json:"roleType"`
	Content string   `json:"content"`
}

// SystemMessage represents a message with Role "system".
func SystemMessage(content string) *Message {
	return &Message{
		Role:    System,
		Content: content,
	}
}

// UserMessage represents a message with Role "user".
func UserMessage(content string) *Message {
	return &Message{
		Role:    User,
		Content: content,
	}
}

func (m *Message) Format(_ context.Context, vs map[string]any, formatType FormatType) ([]*Message, error) {
	c, err := formatContent(m.Content, vs, formatType)
	if err != nil {
		return nil, err
	}

	copied := *m
	copied.Content = c
	return []*Message{&copied}, nil
}

// Format Modified from original source
func formatContent(content string, vs map[string]any, formatType FormatType) (string, error) {
	switch formatType {
	case FString:
		return pyfmt.Fmt(content, vs)
	case GoTemplate:
		parsedTmpl, err := template.New("template").
			Option("missingkey=error").
			Parse(content)
		if err != nil {
			return "", err
		}
		sb := new(strings.Builder)
		err = parsedTmpl.Execute(sb, vs)
		if err != nil {
			return "", err
		}
		return sb.String(), nil
	default:
		return "", fmt.Errorf("unknown format type: %v", formatType)
	}
}

func ConcatMessages(msgs []*Message) (*Message, error) {

	for idx, m := range msgs {
		if m == nil {
			return nil, fmt.Errorf("unexpected nil chunk in message stream, index: %d", idx)
		}
	}

	var (
		contents   []string
		contentLen int
		// toolCalls  []ToolCall
		ret = Message{}
		// extraList = make([]map[string]any, 0, len(msgs))
	)

	for _, msg := range msgs {
		if msg.Role != "" {
			if ret.Role == "" {
				ret.Role = msg.Role
			} else if ret.Role != msg.Role {
				return nil, fmt.Errorf("cannot concat messages with "+
					"different roles: '%s' '%s'", ret.Role, msg.Role)
			}
		}

		// if msg.Name != "" {
		// 	if ret.Name == "" {
		// 		ret.Name = msg.Name
		// 	} else if ret.Name != msg.Name {
		// 		return nil, fmt.Errorf("cannot concat messages with"+
		// 			" different names: '%s' '%s'", ret.Name, msg.Name)
		// 	}
		// }
		//
		// if msg.ToolCallID != "" {
		// 	if ret.ToolCallID == "" {
		// 		ret.ToolCallID = msg.ToolCallID
		// 	} else if ret.ToolCallID != msg.ToolCallID {
		// 		return nil, fmt.Errorf("cannot concat messages with"+
		// 			" different toolCallIDs: '%s' '%s'", ret.ToolCallID, msg.ToolCallID)
		// 	}
		// }

		if msg.Content != "" {
			contents = append(contents, msg.Content)
			contentLen += len(msg.Content)
		}
		//
		// if len(msg.ToolCalls) > 0 {
		// 	toolCalls = append(toolCalls, msg.ToolCalls...)
		// }
		//
		// if len(msg.Extra) > 0 {
		// 	extraList = append(extraList, msg.Extra)
		// }
		//
		// // There's no scenario that requires to concat messages with MultiContent currently
		// if len(msg.MultiContent) > 0 {
		// 	ret.MultiContent = msg.MultiContent
		// }
		//
		// if msg.ResponseMeta != nil && ret.ResponseMeta == nil {
		// 	ret.ResponseMeta = msg.ResponseMeta
		// } else if msg.ResponseMeta != nil && ret.ResponseMeta != nil {
		// 	// keep the last FinishReason with a valid value.
		// 	if msg.ResponseMeta.FinishReason != "" {
		// 		ret.ResponseMeta.FinishReason = msg.ResponseMeta.FinishReason
		// 	}
		//
		// 	if msg.ResponseMeta.Usage != nil {
		// 		if ret.ResponseMeta.Usage == nil {
		// 			ret.ResponseMeta.Usage = &TokenUsage{}
		// 		}
		//
		// 		if msg.ResponseMeta.Usage.PromptTokens > ret.ResponseMeta.Usage.PromptTokens {
		// 			ret.ResponseMeta.Usage.PromptTokens = msg.ResponseMeta.Usage.PromptTokens
		// 		}
		// 		if msg.ResponseMeta.Usage.CompletionTokens > ret.ResponseMeta.Usage.CompletionTokens {
		// 			ret.ResponseMeta.Usage.CompletionTokens = msg.ResponseMeta.Usage.CompletionTokens
		// 		}
		//
		// 		if msg.ResponseMeta.Usage.TotalTokens > ret.ResponseMeta.Usage.TotalTokens {
		// 			ret.ResponseMeta.Usage.TotalTokens = msg.ResponseMeta.Usage.TotalTokens
		// 		}
		//
		// 	}
		//
		// }
	}

	if len(contents) > 0 {
		var sb strings.Builder
		sb.Grow(contentLen)
		sb.WriteString(ret.Content)
		for _, content := range contents {
			_, err := sb.WriteString(content)
			if err != nil {
				return nil, err
			}
		}

		ret.Content = sb.String()
	}

	// if len(toolCalls) > 0 {
	// 	merged, err := concatToolCalls(toolCalls)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	//
	// 	ret.ToolCalls = merged
	// }
	//
	// extra, err := internal.ConcatItems(extraList)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to concat message's extra: %w", err)
	// }
	// if len(extra) > 0 {
	// 	ret.Extra = extra
	// }

	return &ret, nil
}
