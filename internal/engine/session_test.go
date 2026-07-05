// internal/engine/session_test.go
package engine

import (
	"sync"
	"testing"

	"github.com/mambo-wang/go-my-harness/internal/schema"
)

// TestGetWorkingMemory_FullReturn 验证：历史总量不足 limit 时全量返回
func TestGetWorkingMemory_FullReturn(t *testing.T) {
	s := NewSession("test-1", "/tmp")
	s.Append(
		schema.Message{Role: schema.RoleUser, Content: "hello"},
		schema.Message{Role: schema.RoleAssistant, Content: "hi"},
	)

	mem := s.GetWorkingMemory(6)
	if len(mem) != 2 {
		t.Fatalf("期望 2 条，实际 %d 条", len(mem))
	}
	if mem[0].Content != "hello" || mem[1].Content != "hi" {
		t.Error("内容顺序不正确")
	}
}

// TestGetWorkingMemory_Truncation 验证：超过 limit 时从后往前截取最近 N 条
func TestGetWorkingMemory_Truncation(t *testing.T) {
	s := NewSession("test-2", "/tmp")
	// 写入 8 条消息，limit=6 应只返回最后 6 条
	for i := 0; i < 8; i++ {
		s.Append(schema.Message{Role: schema.RoleUser, Content: string(rune('A' + i))})
	}

	mem := s.GetWorkingMemory(6)
	if len(mem) != 6 {
		t.Fatalf("期望 6 条，实际 %d 条", len(mem))
	}
	// 最后 6 条是 C,D,E,F,G,H（第 3~8 条）
	if mem[0].Content != "C" || mem[5].Content != "H" {
		t.Errorf("截取范围错误：首=%s, 末=%s", mem[0].Content, mem[5].Content)
	}
}

// TestGetWorkingMemory_OrphanToolResult 验证：截断后首条是孤儿 ToolResult 时被丢弃
//
// 这是文章中强调的"驾驭防线"：如果截断的第一条恰好是 ToolResult（RoleUser+ToolCallID），
// 但对应的 ToolCall 被截断了，大模型 API 会报 400。GetWorkingMemory 必须丢弃它。
func TestGetWorkingMemory_OrphanToolResult(t *testing.T) {
	s := NewSession("test-3", "/tmp")
	// 构造 8 条历史：
	//  [0] User(A)          ← 被截断
	//  [1] Assistant(B)     ← 被截断
	//  [2] User(ToolResult, ToolCallID="call-1")  ← 截断后首条，孤儿！必须丢弃
	//  [3] User(C)          ← 保留
	//  [4] Assistant(D)
	//  [5] User(E)
	//  [6] Assistant(F)
	//  [7] User(G)
	s.Append(
		schema.Message{Role: schema.RoleUser, Content: "A"},
		schema.Message{Role: schema.RoleAssistant, Content: "B"},
		schema.Message{Role: schema.RoleUser, Content: "tool-output", ToolCallID: "call-1"},
		schema.Message{Role: schema.RoleUser, Content: "C"},
		schema.Message{Role: schema.RoleAssistant, Content: "D"},
		schema.Message{Role: schema.RoleUser, Content: "E"},
		schema.Message{Role: schema.RoleAssistant, Content: "F"},
		schema.Message{Role: schema.RoleUser, Content: "G"},
	)

	mem := s.GetWorkingMemory(6)
	// limit=6 → 截取最后 6 条 = [2..7]，但 [2] 是孤儿 ToolResult，被丢弃
	// 最终应剩 5 条：C, D, E, F, G
	if len(mem) != 5 {
		t.Fatalf("期望 5 条（丢弃孤儿后），实际 %d 条", len(mem))
	}
	if mem[0].Content != "C" {
		t.Errorf("孤儿 ToolResult 未被丢弃，首条内容=%s", mem[0].Content)
	}
	if mem[0].ToolCallID != "" {
		t.Error("截取后首条仍含有 ToolCallID")
	}
}

// TestSessionPhysicalIsolation 验证：两个 Session 的历史完全物理隔离
//
// 对应文章核心场景：群 A 查密钥，群 B 不应看到群 A 的任何消息。
func TestSessionPhysicalIsolation(t *testing.T) {
	mgr := &SessionManager{sessions: make(map[string]*Session)}

	sessA := mgr.GetOrCreate("chat_front_001", "/tmp/project_front")
	sessB := mgr.GetOrCreate("chat_back_002", "/tmp/project_back")

	sessA.Append(
		schema.Message{Role: schema.RoleUser, Content: "帮我看看 README.md 里的密钥"},
		schema.Message{Role: schema.RoleAssistant, Content: "密钥是 token_12345"},
	)

	sessB.Append(
		schema.Message{Role: schema.RoleUser, Content: "别人查到了一个密钥，你这里能看到吗？"},
	)

	// Session B 的工作记忆不应包含 Session A 的任何消息
	memB := sessB.GetWorkingMemory(6)
	if len(memB) != 1 {
		t.Fatalf("Session B 应只有 1 条，实际 %d 条", len(memB))
	}
	for _, m := range memB {
		if m.Content == "密钥是 token_12345" {
			t.Fatal("物理隔离失败：Session B 看到了 Session A 的密钥！")
		}
	}

	// Session A 的工作记忆不应包含 Session B 的消息
	memA := sessA.GetWorkingMemory(6)
	if len(memA) != 2 {
		t.Fatalf("Session A 应有 2 条，实际 %d 条", len(memA))
	}
	for _, m := range memA {
		if m.Content == "别人查到了一个密钥，你这里能看到吗？" {
			t.Fatal("物理隔离失败：Session A 看到了 Session B 的消息！")
		}
	}
}

// TestWorkingMemoryForgetting 验证：长程对话中，超出 limit 的早期消息被"遗忘"
//
// 对应文章核心场景：Session A 塞入 6 条闲聊后，第一轮查到的密钥被 Working Memory 截断丢弃。
func TestWorkingMemoryForgetting(t *testing.T) {
	s := NewSession("chat_front_001", "/tmp/project_front")

	// Turn 1：用户提问 + 助手回答（含密钥）
	s.Append(
		schema.Message{Role: schema.RoleUser, Content: "帮我看看 README.md 里的密钥"},
		schema.Message{Role: schema.RoleAssistant, Content: "密钥是 token_12345"},
	)

	// 塞入 6 条闲聊（3 轮 user+assistant）
	for i := 0; i < 3; i++ {
		s.Append(
			schema.Message{Role: schema.RoleUser, Content: "闲聊占位符"},
			schema.Message{Role: schema.RoleAssistant, Content: "好的，收到闲聊"},
		)
	}

	// Turn 2：用户问密钥
	s.Append(schema.Message{Role: schema.RoleUser, Content: "刚才的密钥是什么？"})

	// limit=6：history 共 9 条，截取最后 6 条
	// 最后 6 条 = [3..8] = 闲聊×6 + 密钥问题
	// 第一轮的密钥回答（index 1）已被截断丢弃
	mem := s.GetWorkingMemory(6)
	if len(mem) != 6 {
		t.Fatalf("期望 6 条，实际 %d 条", len(mem))
	}

	// 验证密钥回答不在工作记忆中
	for _, m := range mem {
		if m.Content == "密钥是 token_12345" {
			t.Fatal("Working Memory 截断失败：早期的密钥回答应被遗忘，但仍出现在工作记忆中")
		}
	}
}

// TestSessionConcurrentSafety 验证：多 goroutine 并发读写 Session 不产生 Data Race
func TestSessionConcurrentSafety(t *testing.T) {
	s := NewSession("concurrent-test", "/tmp")
	var wg sync.WaitGroup

	// 10 个 goroutine 同时追加消息
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			s.Append(schema.Message{Role: schema.RoleUser, Content: "concurrent-msg"})
		}(i)
	}

	// 同时 10 个 goroutine 读取工作记忆
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.GetWorkingMemory(5)
		}()
	}

	wg.Wait()

	// 最终应有 10 条消息
	mem := s.GetWorkingMemory(0) // limit=0 不截断
	if len(mem) != 10 {
		t.Errorf("期望 10 条，实际 %d 条", len(mem))
	}
}

// TestSessionManagerGetOrCreate 验证：同一 ID 返回同一 Session 实例
func TestSessionManagerGetOrCreate(t *testing.T) {
	mgr := &SessionManager{sessions: make(map[string]*Session)}

	sess1 := mgr.GetOrCreate("chat-001", "/tmp/a")
	sess2 := mgr.GetOrCreate("chat-001", "/tmp/a")
	if sess1 != sess2 {
		t.Fatal("同一 ID 应返回同一 Session 实例")
	}

	sess3 := mgr.GetOrCreate("chat-002", "/tmp/b")
	if sess1 == sess3 {
		t.Fatal("不同 ID 应返回不同 Session 实例")
	}
}
