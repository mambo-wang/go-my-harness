// cmd/claw/main.go
//
// 第 12 讲实战：Context Compaction（阶梯降级压缩）防 OOM 验证
//
// 本测试故意给 Agent 挖一个大坑：让它去读取一个含有数万个字符的庞大日志文件
// (mock_log.txt)。read_file 工具内部已截断到 8000 字节，但单条 ToolResult 仍然
// 远超 Compactor 的水位线。我们验证 Context Compactor 的双重降级防线：
//  1. 远期历史的工具输出被整体掩码 (Masking)
//  2. 处于 Working Memory 保护区内的超长工具输出被掐头去尾 (Head-Tail Truncation)
//
// 从而确保发往大模型 API 的上下文永远被限制在安全红线之内，绝不 OOM。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mambo-wang/go-my-harness/internal/engine"
	"github.com/mambo-wang/go-my-harness/internal/provider"
	"github.com/mambo-wang/go-my-harness/internal/schema"
	"github.com/mambo-wang/go-my-harness/internal/tools"
)

func main() {
	// 0. 生成"巨无霸"日志文件，模拟 OOM 场景 (跨平台，不依赖 shell 管道)
	projectRoot, _ := os.Getwd()
	mockLogPath := filepath.Join(projectRoot, "mock_log.txt")
	generateMockLog(mockLogPath)

	// 1. 从 config.json 加载应用配置
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

	// 3. 注册工具，工作区锁定为当前项目根目录
	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool(projectRoot))
	registry.Register(tools.NewWriteFileTool(projectRoot))
	registry.Register(tools.NewBashTool(projectRoot))

	// 4. 创建引擎并唤醒会话
	eng := engine.NewAgentEngine(llmProvider, registry, false)
	reporter := engine.NewTerminalReporter()
	sessionID := "test_oom_protection_001"
	sess := engine.GlobalSessionMgr.GetOrCreate(sessionID, projectRoot)

	// 5. 下达包含多个步骤的连续指令，逼迫 Agent 读取巨型文件
	prompt := `
请帮我执行以下三个步骤：

1. 使用 bash 执行 echo "开始排查日志"
2. 使用 read_file 工具读取当前目录下的巨大文件 mock_log.txt
3. 使用 bash 执行 date 命令获取当前时间，并告诉我任务全部完成。
`

	sess.Append(schema.Message{Role: schema.RoleUser, Content: prompt})
	if err := eng.Run(context.Background(), sess, reporter); err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}
}

// generateMockLog 生成约 100KB 的重复报错日志，用于模拟 OOM 场景
func generateMockLog(path string) {
	const line = "这是一段极其冗长的、无意义的服务器报错日志信息，用来模拟 OOM 场景。\n"
	var sb strings.Builder
	for i := 0; i < 2000; i++ {
		sb.WriteString(line)
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		log.Fatalf("生成 mock_log.txt 失败: %v", err)
	}
	log.Printf("[Setup] 已生成巨无霸日志文件: %s (约 %d 字节)\n", path, sb.Len())
}
