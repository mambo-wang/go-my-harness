package provider

import (
    "context"
    "github.com/mambo-wang/go-my-harness/internal/schema"
)

// LLMProvider 定义了与大模型通信的统一契约
type LLMProvider interface {
    // Generate 接收当前的上下文历史、可用工具列表，并发起一次大模型推理
    Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}