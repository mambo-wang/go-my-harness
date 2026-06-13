package engine

import (
    "context"
    "fmt"
    "log"
    "sync"
    "github.com/mambo-wang/go-my-harness/internal/provider"
    "github.com/mambo-wang/go-my-harness/internal/schema"
    "github.com/mambo-wang/go-my-harness/internal/tools"
)

type AgentEngine struct {
    provider provider.LLMProvider // LLM提供者
    registry tools.Registry // 工具注册表
    WorkDir  string // 工作目录
    EnableThinking bool //慢思考模式开关
}

func NewAgentEngine(p provider.LLMProvider, r tools.Registry, workDir string, enableThinking bool) *AgentEngine { 
    return &AgentEngine{ 
        provider: p, 
        registry: r, 
        WorkDir: workDir,
        EnableThinking: enableThinking, 
    }
}

// Run 方法新增了 Reporter 参数
func (e *AgentEngine) Run(ctx context.Context, userPrompt string, reporter Reporter) error {
    log.Printf("[Engine] 引擎启动，锁定工作区: %s\n", e.WorkDir)

    contextHistory := []schema.Message{
        {Role: schema.RoleSystem, Content: "You are go-my-harness, an expert coding assistant."},
        {Role: schema.RoleUser, Content: userPrompt},
    }

    turnCount := 0

    for {
        turnCount++
        availableTools := e.registry.GetAvailableTools()

        // ================= Phase 1: Thinking =================
        if e.EnableThinking {
            if reporter != nil {
                // 【触发 Reporter】: 开始慢思考
                reporter.OnThinking(ctx)
            }

            thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
            if err != nil {
                return fmt.Errorf("Thinking 生成失败: %w", err)
            }
            if thinkResp.Content != "" {
                contextHistory = append(contextHistory, *thinkResp)
            }
        }

        // ================= Phase 2: Action =================
        actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
        if err != nil {
            return fmt.Errorf("Action 生成失败: %w", err)
        }

        contextHistory = append(contextHistory, *actionResp)

        if actionResp.Content != "" && reporter != nil {
            // 【触发 Reporter】: 输出阶段性总结或最终回复
            reporter.OnMessage(ctx, actionResp.Content)
        }

        // ================= 执行退出与并发控制 =================
        if len(actionResp.ToolCalls) == 0 {
            break
        }

        observationMsgs := make([]schema.Message, len(actionResp.ToolCalls))
        var wg sync.WaitGroup

        for i, toolCall := range actionResp.ToolCalls {
            wg.Add(1)

            go func(idx int, call schema.ToolCall) {
                defer wg.Done()

                if reporter != nil {
                    // 【触发 Reporter】: 报告即将在底层执行的工具
                    reporter.OnToolCall(ctx, call.Name, string(call.Arguments))
                }

                result := e.registry.Execute(ctx, call)

                if reporter != nil {
                    // 为了防止大文件读取导致飞书消息过长被截断，我们仅汇报工具执行状态
                    // 注意：传递给大模型的 observationMsgs 依然是完整数据，只是人类看到的 Reporter 是缩略版
                    displayOutput := result.Output
                    if len(displayOutput) > 200 {
                        displayOutput = displayOutput[:200] + "... (已截断)"
                    }
                    // 【触发 Reporter】: 汇报工具物理执行的结果
                    reporter.OnToolResult(ctx, call.Name, displayOutput, result.IsError)
                }

                observationMsgs[idx] = schema.Message{
                    Role:       schema.RoleUser,
                    Content:    result.Output,
                    ToolCallID: call.ID,
                }
            }(i, toolCall)
        }

        wg.Wait()

        for _, obs := range observationMsgs {
            contextHistory = append(contextHistory, obs)
        }
    }

    return nil
}