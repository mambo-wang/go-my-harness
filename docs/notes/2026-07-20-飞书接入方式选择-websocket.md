---
type: decision
title: "飞书接入方式选择 WebSocket"
date: 2026-07-20
related_modules: ["飞书集成"]
related_components: []
tags: ["decision", "websocket"]
---

## 决策

选择 WebSocket 长连接而非 HTTP 回调方式接入飞书。

## 理由

- 无需公网 IP
- 无需配置回调 URL
- SDK 内置自动重连
- 部署简单，适合内网环境
