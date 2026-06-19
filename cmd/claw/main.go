// cmd/claw/main.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "path/filepath"
    "syscall"

    "github.com/mambo-wang/go-my-harness/internal/engine"
    "github.com/mambo-wang/go-my-harness/internal/feishu"
    "github.com/mambo-wang/go-my-harness/internal/provider"
    "github.com/mambo-wang/go-my-harness/internal/tools"
)

func main() {
    // 1. 获取工作区物理边界
    project_root, _ := os.Getwd()
	workDir := filepath.Join(project_root, "workspace")

    // 2. 从 config.json 配置文件读取应用配置
	configPath := filepath.Join(project_root, "config.json")
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
			if errfeishu := bot.StartWebSocket(ctx); errfeishu != nil {
				log.Printf("❌ WebSocket 连接失败: %v\n", errfeishu)
			}
		}()
	}

    // 【注入新实现的终端输出器】
    reporter := engine.NewTerminalReporter()

    prompt := `
    我需要在当前目录下新建一个 ping.go，提供一个简单的 http ping 接口。
    写完之后，帮我把代码用 git 提交一下。
    `

    err = eng.Run(context.Background(), prompt, reporter)
    if err != nil {
        log.Fatalf("引擎运行崩溃: %v", err)
    }
}