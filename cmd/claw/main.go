// cmd/claw/main.go
//
// 第 11 讲实战：Session 物理隔离与 Working Memory 截取验证
//
// 本测试脚本启动两个 Goroutine，分别模拟飞书"前端群（Session A）"和"后端群（Session B）"，
// 它们同时请求同一个 AgentEngine。我们验证两件事：
//   1. 物理隔离：Session B 看不到 Session A 查到的密钥
//   2. Working Memory 截断：Session A 在塞入 6 条闲聊后，忘掉第一轮查到的密钥
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mambo-wang/go-my-harness/internal/engine"
	"github.com/mambo-wang/go-my-harness/internal/provider"
	"github.com/mambo-wang/go-my-harness/internal/schema"
	"github.com/mambo-wang/go-my-harness/internal/tools"
)

func main() {
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

	// 3. 准备测试环境：创建两个模拟项目目录
	os.MkdirAll("/tmp/project_front", 0755)
	os.MkdirAll("/tmp/project_back", 0755)
	os.WriteFile("/tmp/project_front/README.md",
		[]byte("这是项目 A 的 README，里面包含了一个密钥: token_12345"), 0644)

	// 4. 创建引擎（Registry 绑定到 project_front）
	//    注意：EnableThinking 关闭，聚焦测试 Session 隔离本身
	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool("/tmp/project_front"))
	eng := engine.NewAgentEngine(llmProvider, registry, false)

	reporter := engine.NewTerminalReporter()

	// 5. 并发测试：两个 Goroutine 模拟飞书"前端群"和"后端群"
	var wg sync.WaitGroup

	// Session A: 前端群 — 测试 Working Memory 截断
	wg.Add(1)
	go func() {
		defer wg.Done()
		sessionA := engine.GlobalSessionMgr.GetOrCreate("chat_front_001", "/tmp/project_front")

		log.Println("\n>>> 🙋 [Session A / Turn 1]: 帮我看看 README.md 里记录了什么密钥？")
		sessionA.Append(schema.Message{Role: schema.RoleUser, Content: "帮我看看 README.md 里记录了什么密钥？"})
		_ = eng.Run(context.Background(), sessionA, reporter)

		// 塞入 6 条闲聊，逼迫 Working Memory（limit=6）截断第一轮的 ToolResult
		for i := 0; i < 6; i++ {
			sessionA.Append(schema.Message{Role: schema.RoleUser, Content: "这只是一句闲聊占位符。"})
			sessionA.Append(schema.Message{Role: schema.RoleAssistant, Content: "好的，收到闲聊。"})
		}

		log.Println("\n>>> 🙋 [Session A / Turn 2]: 请直接告诉我，刚才第一轮你查到的那个密钥是什么？")
		sessionA.Append(schema.Message{Role: schema.RoleUser, Content: "请直接告诉我，刚才第一轮你查到的那个密钥是什么？不准调用工具！"})
		_ = eng.Run(context.Background(), sessionA, reporter)
	}()

	// Session B: 后端群 — 测试物理隔离
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Second) // 让 Session A 先跑一会

		sessionB := engine.GlobalSessionMgr.GetOrCreate("chat_back_002", "/tmp/project_back")

		log.Println("\n>>> 🙋 [Session B]: 别人查到了一个密钥，你这里能看到吗？")
		sessionB.Append(schema.Message{Role: schema.RoleUser, Content: "别人查到了一个密钥，你这里能看到吗？不准调用工具！"})
		_ = eng.Run(context.Background(), sessionB, reporter)
	}()

	wg.Wait()
	log.Println("\n✅ 并发测试完成")
}
