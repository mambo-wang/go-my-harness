---
type: decision
title: "双协议 LLM Provider 抽象层"
date: 2026-06-28
related_modules: ["Provider", "Engine"]
related_components: []
tags: ["decision", "deepseek", "maxtokens", "minimaxprovider"]
---

## 背景

不同 LLM 供应商（OpenAI、Anthropic）的 API 协议差异很大，直接耦合会导致切换成本极高。

## 决策

定义统一的 LLMProvider 接口，两个内置实现：
- **OpenAIProvider**：基于 openai-go SDK，兼容 OpenAI 协议（用于 DeepSeek 等）
- **MiniMaxProvider**：基于 anthropic-sdk-go，兼容 Anthropic 协议

关键差异处理：
- System 消息位置（OpenAI 放 messages 数组，Anthropic 用独立 system 字段）
- 工具结果回传格式不同
- Assistant 工具调用格式不同
- Anthropic 要求 MaxTokens 必填

## 影响

Engine 通过接口与 Provider 解耦，新增模型供应商只需实现 LLMProvider 接口，无需修改引擎代码。配置系统（config.json）通过 provider 字段路由创建对应实现。
