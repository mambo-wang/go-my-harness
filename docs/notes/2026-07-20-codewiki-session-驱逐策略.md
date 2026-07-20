---
type: pitfall
title: "CodeWiki Session 驱逐策略"
date: 2026-07-20
related_modules: ["引擎核心"]
related_components: []
tags: ["codewiki", "pitfall"]
aliases: ["session驱逐", "LRU"]
severity: medium
root_cause: "Session LRU 容量限制为 10，子代理独立创建 session 会耗尽配额"
---

## 问题描述

CodeWiki MCP 最多维护 10 个 session，超出后静默驱逐最久未访问的 session。如果子代理自行调用 analyze_repo 创建新 session，可能导致主代理 session 被驱逐。

## 根因

[Session](../internal/engine/session.go) 设计为 LRU 缓存，容量上限 10。多子代理并发时各自创建 session 会快速耗尽配额。

## 解决方案

子代理必须共享主代理的 session_id，不得自行调用 analyze_repo。已在 SKILL.md 中明确约束。
