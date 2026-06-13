# Go 语法速查表（基于 go-my-harness 项目）

## 1. 包与导入

```go
package provider  // 声明当前文件属于 provider 包

import (
    "context"                                        // 标准库
    "encoding/json"                                   // 标准库
    "fmt"                                            // 标准库
    "os"                                             // 标准库

    "github.com/anthropics/anthropic-sdk-go"         // 第三方库
    "github.com/mambo-wang/go-my-harness/internal/schema"  // 自定义内部包
)
```

- 同一包内的文件可以互相访问（无需 import）
- 外部包使用 `包名.符号名` 访问

---

## 2. 类型声明

### 2.1 结构体 (Struct)

```go
type Message struct {
    Role       Role           `json:"role"`           // json tag 用于序列化
    Content    string         `json:"content"`
    ToolCalls  []ToolCall     `json:"tool_calls,omitempty"` // omitempty: 空时省略
    ToolCallID string         `json:"tool_call_id,omitempty"`
}
```

**struct tag 说明：**
| tag | 含义 |
|-----|------|
| `json:"fieldName"` | 序列化时的字段名 |
| `json:"-"` | 忽略该字段 |
| `json:",omitempty` | 空值时省略字段 |

### 2.2 类型别名

```go
type Role string  // Role 就是 string 的别名
```

### 2.3 常量组

```go
const (
    RoleSystem    Role = "system"    // 系统提示词
    RoleUser      Role = "user"      // 用户输入
    RoleAssistant Role = "assistant" // 助手输出
)
```

### 2.4 空接口 (任意类型)

```go
InputSchema interface{}  // 可以接受任意类型，类似 Java 的 Object
```

---

## 3. 接口 (Interface)

```go
type LLMProvider interface {
    Generate(ctx context.Context, messages []schema.Message,
             availableTools []schema.ToolDefinition) (*schema.Message, error)
}
```

**Go 的接口是隐式实现**：只要某个类型实现了接口声明的所有方法，就自动满足该接口，无需 `implements` 关键字。

---

## 4. 方法与接收者 (Receiver)

```go
// 值接收者
func (m *MockProvider) Generate(...) (*schema.Message, error) { ... }

// 等价于
func (m MockProvider) Generate(...) (*schema.Message, error) { ... }
```

**值接收者 vs 指针接收者：**

| 形式 | 说明 |
|------|------|
| `(p *OpenAIProvider)` | 指针接收者，可以修改原对象，效率更高（不拷贝） |
| `(p OpenAIProvider)` | 值接收者，拿到的是副本，无法修改原对象 |

**常见约定**：如果方法可能修改对象、或结构体较大，通常用指针接收者。

---

## 5. 指针基础

```go
p := &OpenAIProvider{}   // & 取地址，得到指针
v := *p                  // * 解引用，通过指针访问实际对象
```

| 符号 | 用法 |
|------|------|
| `*Type` | 指向 Type 的指针类型 |
| `&var` | 获取变量地址 |
| `*ptr` | 解引用，获取指针指向的值 |

---

## 6. 切片 (Slice)

```go
// 切片声明
var msgs []schema.Message
msgs := []schema.Message{}

// 追加元素
msgs = append(msgs, msg)

// 子切片
someSlice[1:3]  // 索引 1 到 2（不包含 3）
```

切片是引用类型，类似于动态数组。

---

## 7. Map (字典)

```go
// 声明
properties := map[string]any{}

// 赋值
properties["name"] = "string"
properties["age"] = 18

// 访问
val, ok := properties["name"]  // ok 表示是否存在
```

---

## 8. 错误处理

```go
// 返回 error
func (p *OpenAIProvider) Generate(...) (*schema.Message, error) {
    if err != nil {
        return nil, fmt.Errorf("请求失败: %w", err)  // %w 包装错误
    }
    return result, nil
}

// 调用时检查
resp, err := p.Generate(ctx, msgs, tools)
if err != nil {
    log.Fatalf("引擎崩溃: %v", err)
}
```

**注意**：Go 没有异常机制，错误通过返回值传递。

---

## 9. Context (上下文)

```go
import "context"

func Generate(ctx context.Context, ...) (*schema.Message, error) {
    // 用于：超时控制、取消请求、传递请求级数据
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    // ...
}
```

---

## 10. JSON 处理

```go
import "encoding/json"

// 序列化
data, _ := json.Marshal(v)

// 反序列化
var m map[string]interface{}
_ = json.Unmarshal(data, &m)

// json.RawMessage 延迟解析（不预解析为具体类型）
type ToolCall struct {
    Arguments json.RawMessage `json:"arguments"`  // 保持为原始字节
}
```

**json.RawMessage**：适合不确定结构、或需要延迟解析的场景。

---

## 11. 类型断言

```go
// Interface{} 转具体类型
if m, ok := toolDef.InputSchema.(map[string]interface{}); ok {
    // 转换成功，ok 为 true
}
```

**语法**：`value.(Type)`，返回值可以有一个或两个（ok bool）。

---

## 12. 匿名函数与闭包

```go
// 直接调用
func() {
    fmt.Println("立即执行")
}()

// 赋值给变量
add := func(a, b int) int {
    return a + b
}
result := add(1, 2)
```

---

## 13. defer 延迟执行

```go
func readFile(name string) {
    f, _ := os.Open(name)
    defer f.Close()  // 函数退出前自动执行
    // ... 读取文件
}
```

---

## 14. 初始化与构造函数

```go
// 构造函数约定：New 开头
func NewOpenAIProvider(model string) *OpenAIProvider {
    return &OpenAIProvider{
        client: openai.NewClient(...),
        model:  model,
    }
}
```

---

## 15. 包的 GOPATH 布局

```
go-my-harness/
├── cmd/claw/main.go              // 应用入口
└── internal/
    ├── engine/loop.go            // internal 包，仅同模块内可访问
    ├── provider/interface.go     // provider 子包
    └── schema/message.go         // schema 子包
```

| 目录 | 说明 |
|------|------|
| `cmd/` | 可执行程序入口 |
| `internal/` | 内部包，仅同模块引用 |
| `pkg/` | 公开可复用的包（若有） |

---

## 16. 类型 switch

```go
switch block.Type {
case "text":
    resultMsg.Content += block.Text
case "tool_use":
    // 处理工具调用
}
```

---

## 17. 快速参考

| 语法 | 说明 |
|------|------|
| `var x int` | 声明变量（零值） |
| `x := 1` | 短变量声明 |
| `const X = 1` | 常量 |
| `make([]Type, n)` | 创建切片 |
| `make(map[K]V)` | 创建 Map |
| `len(slice)` | 长度 |
| `cap(slice)` | 容量 |
| `range slice` | 遍历 |
| `go func()` | 启动 Goroutine |
| `<-ch` | 接收 Channel |

---

## 18. 本项目核心接口速查

```go
// Provider 接口：负责与大模型通信
type LLMProvider interface {
    Generate(ctx context.Context, messages []schema.Message,
             availableTools []schema.ToolDefinition) (*schema.Message, error)
}

// Registry 接口：负责工具注册与执行
type Registry interface {
    GetAvailableTools() []schema.ToolDefinition
    Execute(ctx context.Context, call schema.ToolCall) schema.ToolResult
}
```