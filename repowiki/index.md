# 项目文档索引

> 自动生成于 2026-07-05T20:50:05+08:00 | 本文件由系统自动维护

## 模块文档

| 文档 | 说明 |
|------|------|
| [go-my-harness 仓库总览](overview.md) | go-my-harness 是一个基于 Go 语言开发的 AI Agent 框架，采用“驾驭工程”（Harness Engineering）理念构建。项目实现了完整的 Agent 运行循环——思考(Think) → 行动(Act) → 观察 |
| [LLM 提供者模块](LLM提供者.md) | LLM 提供者模块是 go-my-harness 框架中负责对接大语言模型（LLM）推理服务的核心抽象层。它通过定义统一的 `LLMProvider` 接口，将不同 LLM 厂商的 API 差异封装在具体实现中，使上层 Agent 循环可以 |
| [上下文管理模块](上下文管理.md) | 上下文管理模块（`internal/context`）是 go-my-harness AI Agent 框架的**提示词组装引擎**，负责在每次 Agent 调用前，将分散在不同来源的上下文信息动态聚合为一条完整的系统提示词（System  |
| [工具系统 (Tools System)](工具系统.md) | 工具系统是 go-my-harness AI Agent 框架的核心基础设施模块，位于 `internal/tools/` 包下。它为大语言模型 (LLM) 提供了一套与物理世界交互的标准化工具集合，使 AI Agent 能够执行 Bash |
| [应用入口](应用入口.md) | 应用入口模块是 go-my-harness 项目的启动层，提供了三种不同的运行模式入口：CLI 命令行模式（`cmd/claw/main.go`）、HTTP 服务器模式（`server.go`）和最小可运行示例（`helloworld.go |
| [引擎核心](引擎核心.md) | 引擎核心（`internal/engine`）是 go-my-harness AI Agent 框架的心脏模块。它实现了 Agent 的主运行循环——**思考(Think) → 行动(Act) → 观察(Observe)** 循环，并负责会 |
| [飞书集成模块](飞书集成.md) | 飞书集成模块（`internal/feishu/bot.go`）是 go-my-harness 项目中负责与飞书（Lark）平台对接的核心适配层。该模块将飞书机器人接收到的用户消息桥接到内部的 AI Agent 引擎，并将 Agent 的思 |

## 知识笔记

| 标题 | 类型 | 日期 | 文件 |
|------|------|------|------|
