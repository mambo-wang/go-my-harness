// MiniMax Claude Format Provider
// 基于 Anthropic SDK，指向 MiniMax Claude Format API
package provider

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
    "github.com/mambo-wang/go-my-harness/internal/schema"
)

type MiniMaxProvider struct {
    client anthropic.Client
    model  string
}

// NewAnthropicProvider 通用构造函数：接受 apiKey、baseURL 和模型名称
func NewAnthropicProvider(apiKey, baseURL, model string) *MiniMaxProvider {
    return &MiniMaxProvider{
        client: anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL)),
        model:  model,
    }
}

func (p *MiniMaxProvider) Generate(ctx context.Context, msgs []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error) {
    var anthropicMsgs []anthropic.MessageParam
    var systemPrompt string

    // 1. 消息翻译
    for _, msg := range msgs {
        switch msg.Role {
        case schema.RoleSystem:
            systemPrompt = msg.Content
        case schema.RoleUser:
            if msg.ToolCallID != "" {
                anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
                    anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false),
                ))
            } else {
                anthropicMsgs = append(anthropicMsgs, anthropic.NewUserMessage(
                    anthropic.NewTextBlock(msg.Content),
                ))
            }
        case schema.RoleAssistant:
            var blocks []anthropic.ContentBlockParamUnion
            if msg.Content != "" {
                blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
            }

            // 将历史工具调用转回 Claude 特有的 ToolUseBlockParam
            for _, tc := range msg.ToolCalls {
                var inputMap map[string]interface{}
                _ = json.Unmarshal(tc.Arguments, &inputMap)
                blocks = append(blocks, anthropic.ContentBlockParamUnion{
                    OfToolUse: &anthropic.ToolUseBlockParam{
                        ID:    tc.ID,
                        Name:  tc.Name,
                        Input: inputMap,
                    },
                })
            }
            if len(blocks) > 0 {
                anthropicMsgs = append(anthropicMsgs, anthropic.NewAssistantMessage(blocks...))
            }
        }
    }

    // 2. 工具 Schema 翻译
    var anthropicTools []anthropic.ToolUnionParam
    for _, toolDef := range availableTools {
        // ToolInputSchemaParam 是结构体，需要通过 Properties 字段精准填充
        var properties map[string]any
        var required []string

        if m, ok := toolDef.InputSchema.(map[string]interface{}); ok {
            if p, ok := m["properties"].(map[string]interface{}); ok {
                properties = p
            }
            if r, ok := m["required"].([]string); ok {
                required = r
            }
        }

        tp := anthropic.ToolParam{
            Name:        toolDef.Name,
            Description: anthropic.String(toolDef.Description),
            InputSchema: anthropic.ToolInputSchemaParam{
                Properties: properties,
                Required:   required,
            },
        }
        anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{OfTool: &tp})
    }

    // 3. 构建请求并发送
    params := anthropic.MessageNewParams{
        Model:     anthropic.Model(p.model),
        MaxTokens: 4096,
        Messages:  anthropicMsgs,
    }

    if systemPrompt != "" {
        params.System = []anthropic.TextBlockParam{
            {Text: systemPrompt},
        }
    }

    if len(anthropicTools) > 0 {
        params.Tools = anthropicTools
    }

    resp, err := p.client.Messages.New(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("Claude/MiniMax API 请求失败: %w", err)
    }

    // 4. 反向解析
    resultMsg := &schema.Message{
        Role: schema.RoleAssistant,
    }

    for _, block := range resp.Content {
        switch block.Type {
        case "text":
            resultMsg.Content += block.Text
        case "tool_use":
            argsBytes, _ := json.Marshal(block.Input)
            resultMsg.ToolCalls = append(resultMsg.ToolCalls, schema.ToolCall{
                ID:        block.ID,
                Name:      block.Name,
                Arguments: argsBytes,
            })
        }
    }

    return resultMsg, nil
}