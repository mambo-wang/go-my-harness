# 项目文档索引

> 自动生成于 2026-06-28T22:00:43+08:00 | 本文件由系统自动维护

## 模块文档

| 文档 | 说明 |
|------|------|
| [go-my-harness 项目总览](overview.md) | **go-my-harness** 是一个基于 Go 语言构建的 AI Agent 框架，实现了经典的 **ReAct（Reasoning + Acting）** 循环模式。项目的核心理念是"驾驭工程"（Harness Engineerin |
| [Agent引擎](Engine.md) | Agent引擎是 go-my-harness 的核心驱动力，实现了 ReAct（Reasoning + Acting）循环模式。它协调 LLM Provider、工具注册表和 [Reporter](../internal/engine/re |
| [Prompt上下文](Context.md) | Prompt上下文模块负责动态组装 Agent 的 System Prompt。它将极简内核指令、项目专属规范和外部技能模板三层内容融合为一条完整的系统消息，使 Agent 能够适应不同的工作区和技能配置。 |
| [Provider 模块 — LLM 提供者层](Provider.md) | Provider 模块是 `go-my-harness` 系统中负责与外部大语言模型（LLM）进行通信的抽象层。它定义了一套统一的接口契约 `LLMProvider`，使得上层引擎（[Engine](Engine.md)）无需关心底层模型的 |
| [工具系统](Tools.md) | 工具系统是 go-my-harness 的执行层，实现了基于接口契约的工具注册、路由和执行机制。四个具体工具（Bash、ReadFile、WriteFile、EditFile）通过统一的 [Registry](../internal/too |
| [应用入口](Entry.md) | 应用入口模块包含了 go-my-harness 项目的所有可执行程序入口。项目提供了三种运行模式：CLI Agent 主程序、HTTP 服务器和 Ping 健康检查服务。 |
| [数据模型](Schema.md) | 数据模型模块定义了 go-my-harness 系统中所有与 LLM 交互的核心数据结构。这些结构是 Agent 引擎、LLM Provider 和工具系统之间的通用语言，确保各模块对消息格式的理解一致。 |
| [飞书集成模块](Feishu.md) | 飞书集成模块是系统与飞书（Lark）平台之间的桥梁层，负责接收飞书消息并驱动 Agent 引擎执行任务，同时将执行过程与结果实时回传至飞书会话。该模块基于飞书官方 SDK 实现，支持 WebSocket 长连接和 HTTP 回调两种事件接入 |

## 知识笔记

| 标题 | 类型 | 日期 | 文件 |
|------|------|------|------|
| ReAct 循环模式作为核心 Agent 架构 | decision | 2026-06-28 | [链接](notes/2026-06-28-react-循环模式作为核心-agent-架构.md) |
| 双协议 LLM Provider 抽象层 | decision | 2026-06-28 | [链接](notes/2026-06-28-双协议-llm-provider-抽象层.md) |
| 四层安全护栏设计 | decision | 2026-06-28 | [链接](notes/2026-06-28-四层安全护栏设计.md) |
