package abstract

import (
	"context"
	"llm-util/ai/app/abstract/schema"
)

type App interface {
	Generate(ctx context.Context, input []*schema.Message) (*schema.Message, error)
	Stream(ctx context.Context, input []*schema.Message) (*schema.StreamReader[*schema.Message], error)
}
