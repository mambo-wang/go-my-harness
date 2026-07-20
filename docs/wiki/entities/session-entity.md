---
type: Entity
title: Session
description: "会话状态容器，承载 Agent 运行时的完整上下文历史"
aliases: ["会话", "SessionManager", "GlobalSessionMgr"]
resource: file://internal/engine/session.go
tags: [go-my-harness, 引擎核心]
---

# [Session](../../../internal/engine/session.go)

## 概述

`Session` 是 go-my-harness 中的会话状态容器，采用“无状态引擎 + 有状态 [Session](../../../internal/engine/session.go)”架构，实现多端并发下的物理隔离。

## 核心结构

```go
type Session struct {
    ID        string
    WorkDir   string
    history   []schema.Message
    mu        sync.RWMutex
}
```

## 关键方法

| 方法 | 职责 |
|------|------|
| `Append` | 线程安全追加消息 |
| `GetWorkingMemory` | 截取最近 N 条消息作为短期记忆 |

## 设计要点

- 读写锁保护并发安全
- 孤儿 [ToolResult](../../../internal/schema/message.go) 修复：截断点落在工具响应时自动跳过
- [SessionManager](../../../internal/engine/session.go) 以 map 管理多会话，ChatID 作为隔离键
