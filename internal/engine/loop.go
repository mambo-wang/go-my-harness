// internal/engine/loop.go
package engine

import (
    "context"
    "fmt"
    "log"
    "sync"

    ctxpkg "github.com/mambo-wang/go-my-harness/internal/context"
    "github.com/mambo-wang/go-my-harness/internal/provider"
    "github.com/mambo-wang/go-my-harness/internal/schema"
    "github.com/mambo-wang/go-my-harness/internal/tools"
)

// AgentEngine 是 Agent 的核心引擎。它本身不维护任何状态，
// 而是依靠传入的 Session 实例作为上下文的承载体，实现多端并发下的物理隔离。
type AgentEngine struct {
    provider       provider.LLMProvider // LLM 提供者
    registry       tools.Registry       // 工具注册表
    EnableThinking bool                 // 慢思考模式开关
}

// NewAgentEngine 创建引擎实例。
// 注意：不再接收 workDir 参数——工作区由每次 Run 时传入的 Session 决定。
func NewAgentEngine(p provider.LLMProvider, r tools.Registry, enableThinking bool) *AgentEngine {
    return &AgentEngine{
        provider:       p,
        registry:       r,
        EnableThinking: enableThinking,
    }
}

// Run 以传入的 Session 作为上下文承载体，执行 Agent 循环。
// 引擎不再"用完即毁"，而是从 Session 中恢复记忆，执行完毕后将新产生的消息追加回 Session。
func (e *AgentEngine) Run(ctx context.Context, session *Session, reporter Reporter) error {
    log.Printf("[Engine] 唤醒会话 [%s]，锁定工作区: %s\n", session.ID, session.WorkDir)

    // 根据 Session 绑定的工作区动态组装 System Prompt
    composer := ctxpkg.NewPromptComposer(session.WorkDir)
    systemMsg := composer.Build()

    for {
        availableTools := e.registry.GetAvailableTools()

        // 【核心】Working Memory 截取：不返回全量历史，只取最近 N 条作为短期工作记忆
        workingMemory := session.GetWorkingMemory(6)

        // 每次循环都重新拼装 contextHistory：System Prompt + Working Memory
        var contextHistory []schema.Message
        contextHistory = append(contextHistory, systemMsg)
        contextHistory = append(contextHistory, workingMemory...)

        // ================= Phase 1: Thinking =================
        if e.EnableThinking {
            if reporter != nil {
                reporter.OnThinking(ctx)
            }
            thinkResp, err := e.provider.Generate(ctx, contextHistory, nil)
            if err != nil {
                return fmt.Errorf("Thinking 阶段失败: %w", err)
            }
            if thinkResp.Content != "" {
                session.Append(*thinkResp)
                contextHistory = append(contextHistory, *thinkResp)
            }
        }

        // ================= Phase 2: Action =================
        actionResp, err := e.provider.Generate(ctx, contextHistory, availableTools)
        if err != nil {
            return fmt.Errorf("Action 阶段失败: %w", err)
        }
        session.Append(*actionResp)
        contextHistory = append(contextHistory, *actionResp)

        if actionResp.Content != "" && reporter != nil {
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
                    reporter.OnToolCall(ctx, call.Name, string(call.Arguments))
                }

                result := e.registry.Execute(ctx, call)

                if reporter != nil {
                    displayOutput := result.Output
                    if len(displayOutput) > 200 {
                        displayOutput = displayOutput[:200] + "... (已截断)"
                    }
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

        // 工具执行结果追加回 Session，供下一轮 Working Memory 截取
        session.Append(observationMsgs...)
    }

    return nil
}
