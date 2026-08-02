// cmd/claw/main.go
//
// 第 13 讲实战：基于文件系统的持久化记忆与 Plan Mode (状态外部化)
//
// 本讲核心：通过 -prompt 命令行参数接收人类指令，多次独立唤醒 Agent。
// 当开启 PlanMode 时，引擎会在 System Prompt 中注入"长程任务与状态外部化
// 强制规范"，引导 Agent 将宏观架构写入 PLAN.md、将待办清单写入 TODO.md，
// 并在每步执行后立即打勾。进程重启后 Agent 通过 read_file 这两个文件即可
// 实现"断点续传"，人类也可随时手动编辑文件进行纠偏 (Human-in-the-loop)。

// 同时抹平了严格 OpenAI 兼容端点的一个隐蔽坑：当 assistant 携带 tool_calls
// 但 content 为空时，也必须显式传递 "" 字段，否则会触发 400 Bad Request。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mambo-wang/go-my-harness/internal/engine"
	"github.com/mambo-wang/go-my-harness/internal/provider"
	"github.com/mambo-wang/go-my-harness/internal/schema"
	"github.com/mambo-wang/go-my-harness/internal/tools"
)

func main() {
	// 0. 通过命令行参数接收人类指令 (支持多次独立唤醒)
	promptPtr := flag.String("prompt", "", "要交给 Agent 执行的任务描述")
	flag.Parse()
	if *promptPtr == "" {
		fmt.Println("用法: go run cmd/claw/main.go -prompt \"你的任务指令\"")
		os.Exit(1)
	}

	// 1. 从 config.json 加载应用配置
	projectRoot, _ := os.Getwd()
	configPath := filepath.Join(projectRoot, "config.json")
	cfg, err := provider.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	mc, modelName, err := cfg.GetModelConfig("")
	if err != nil {
		log.Fatalf("获取模型配置失败: %v", err)
	}
	fmt.Printf("[Config] 使用模型: %s (provider=%s)\n", modelName, mc.Provider)

	// 2. 根据 provider 类型创建 LLM Provider
	var llmProvider provider.LLMProvider
	switch mc.Provider {
	case "openai":
		llmProvider = provider.NewOpenAIProvider(mc.APIKey, mc.BaseURL, modelName)
	case "anthropic":
		llmProvider = provider.NewAnthropicProvider(mc.APIKey, mc.BaseURL, modelName)
	default:
		log.Fatalf("不支持的 provider 类型: %s", mc.Provider)
	}

	// 3. 注册工具，工作区锁定为当前项目根目录 (含 PLAN.md / TODO.md 读写能力)
	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool(projectRoot))
	registry.Register(tools.NewWriteFileTool(projectRoot))
	registry.Register(tools.NewBashTool(projectRoot))
	registry.Register(tools.NewEditFileTool(projectRoot))

	// 4. 创建引擎 (开启 PlanMode，关闭慢思考以追求演示效果) 并唤醒会话
	eng := engine.NewAgentEngine(llmProvider, registry, false, true)
	reporter := engine.NewTerminalReporter()
	sessionID := "task_web_server_01"
	sess := engine.GlobalSessionMgr.GetOrCreate(sessionID, projectRoot)

	// 5. 下达任务指令，开始文件系统的持久化记忆之旅
	log.Printf("\n>>> 🚀 收到指令: %s\n", *promptPtr)
	sess.Append(schema.Message{Role: schema.RoleUser, Content: *promptPtr})

	if err := eng.Run(context.Background(), sess, reporter); err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}
