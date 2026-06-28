---
type: decision
title: "ReAct 循环模式作为核心 Agent 架构"
date: 2026-06-28
related_modules: ["Engine", "Tools"]
related_components: []
tags: ["decision", "react", "waitgroup"]
---

## 背景

go-my-harness 是一个 Go 语言的 AI Agent 框架，核心目标是安全地驾驭 LLM 与真实世界交互。

## 决策

采用 ReAct（Reasoning + Acting）循环模式作为 Agent 核心架构：
1. **Thinking 阶段**：可选的纯推理轮，让 LLM 先思考再行动
2. **Action 阶段**：LLM 决策是否调用工具
3. **Observation 阶段**：并发执行工具（sync.WaitGroup），结果反馈给 LLM

## 备选方案

- **Plan-and-Execute**：先制定完整计划再执行 — 放弃，因为 LLM 的计划能力有限，且无法处理执行中的动态变化
- **纯 Tool-Use Loop**（无 Thinking）：跳过推理直接工具调用 — 放弃，因为在复杂任务上成功率较低

## 影响

Engine 模块成为系统核心，所有其他模块围绕 ReAct 循环设计。Reporter 接口抽象了输出通道，使得同一引擎可以同时支持终端和飞书 IM。
