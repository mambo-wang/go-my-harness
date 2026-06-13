// cmd/claw/main.go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "path/filepath"
    "strings"
    "syscall"

    "github.com/mambo-wang/go-my-harness/internal/engine"
    "github.com/mambo-wang/go-my-harness/internal/feishu"
    "github.com/mambo-wang/go-my-harness/internal/provider"
    "github.com/mambo-wang/go-my-harness/internal/tools"
)

func main() {
    // 1. 获取工作区物理边界
    workDir, _ := os.Getwd()

    // 2. 从 config.json 配置文件读取应用配置
    configPath := filepath.Join(workDir, "config.json")
    cfg, err := provider.LoadConfig(configPath)
    if err != nil {
        log.Fatalf("加载配置文件失败: %v", err)
    }

    mc, modelName, err := cfg.GetModelConfig("")
    if err != nil {
        log.Fatalf("获取模型配置失败: %v", err)
    }
    fmt.Printf("[Config] 使用模型: %s (provider=%s)\n", modelName, mc.Provider)

    // 3. 根据 provider 类型创建 LLM Provider
    var llmProvider provider.LLMProvider
    switch mc.Provider {
    case "openai":
        llmProvider = provider.NewOpenAIProvider(mc.APIKey, mc.BaseURL, modelName)
    case "anthropic":
        llmProvider = provider.NewAnthropicProvider(mc.APIKey, mc.BaseURL, modelName)
    default:
        log.Fatalf("不支持的 provider 类型: %s", mc.Provider)
    }

    registry := tools.NewRegistry()
    registry.Register(tools.NewReadFileTool(workDir))
    registry.Register(tools.NewWriteFileTool(workDir))
    registry.Register(tools.NewBashTool(workDir))
    registry.Register(tools.NewEditFileTool(workDir))

	eng := engine.NewAgentEngine(llmProvider, registry, workDir, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 飞书模式：配置文件中飞书字段非空时后台启动
	if cfg.Feishu != nil && cfg.Feishu.AppID != "" && cfg.Feishu.AppSecret != "" {
		bot := feishu.NewFeishuBot(eng, cfg.Feishu)
		go func() {
			log.Println("🚀 飞书 WebSocket 长连接模式启动...")
			if err := bot.StartWebSocket(ctx); err != nil {
				log.Printf("❌ WebSocket 连接失败: %v\n", err)
			}
		}()
	}

	// 终端交互模式：始终启动
	fmt.Println("🖥️  Go My Harness终端模式 (输入 exit 或 quit 退出)")
	fmt.Println("─────────────────────────────────────────────────")

	reporter := engine.NewTerminalReporter()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")

		inputCh := make(chan string, 1)
		go func() {
			if scanner.Scan() {
				inputCh <- scanner.Text()
			} else {
				inputCh <- ""
			}
		}()

		select {
		case <-sigChan:
			fmt.Println("\n📴 再见！")
			cancel()
			return
		case input := <-inputCh:
			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}
			if input == "exit" || input == "quit" {
				fmt.Println("📴 再见！")
				cancel()
				return
			}

			runCtx, runCancel := context.WithCancel(ctx)
			done := make(chan struct{})

			go func() {
				defer close(done)
				if err := eng.Run(runCtx, input, reporter); err != nil && runCtx.Err() == nil {
					log.Printf("❌ Agent 运行失败: %v\n", err)
				}
			}()

			select {
			case <-done:
				runCancel()
			case <-sigChan:
				runCancel()
				<-done
				fmt.Println("\n📴 再见！")
				cancel()
				return
			}
		}
	}
}