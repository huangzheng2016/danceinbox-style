package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"
)

// DemoAgent stands in for a model call so the streaming path can be developed
// and tested without one: it streams Markdown in small chunks and makes two tool
// calls, the second failing when the example's remark contains "fail".
type DemoAgent struct {
	q *query.Query
	// pause between text chunks
	Step time.Duration
}

func NewDemoAgent(q *query.Query) Agent { return &DemoAgent{q: q, Step: 40 * time.Millisecond} }

func (a *DemoAgent) Run(ctx context.Context, ex *model.Example, out Output) error {
	if err := a.say(ctx, out, fmt.Sprintf("正在检查示例 **%s**。\n\n", ex.Name)); err != nil {
		return err
	}
	stats, err := out.Tool(ctx, "count_examples", map[string]any{"status": "enabled"}, func(ctx context.Context) (any, error) {
		e := a.q.Example
		n, err := e.WithContext(ctx).Where(e.Status.Eq(model.StatusEnabled)).Count()
		return map[string]int64{"enabled": n}, err
	})
	if err != nil {
		return err
	}
	_, err = out.Tool(ctx, "check_config", map[string]any{"example_id": ex.ID}, func(ctx context.Context) (any, error) {
		if err := sleep(ctx, 600*time.Millisecond); err != nil {
			return nil, err
		}
		if strings.Contains(ex.Remark, "fail") {
			return nil, errors.New("配置校验失败:缺少必填项 target")
		}
		cfg := map[string]any{}
		if ex.Config != "" {
			if err := json.Unmarshal([]byte(ex.Config), &cfg); err != nil {
				return nil, err
			}
		}
		return map[string]any{"valid": true, "keys": len(cfg)}, nil
	})
	if err != nil {
		return err
	}
	return a.say(ctx, out, fmt.Sprintf(
		"检查完成：\n\n| 项目 | 结果 |\n| --- | --- |\n| 已启用示例 | %d |\n| 配置 | 有效 |\n\n下一步可以在 `配置` 中调整参数后再次运行。",
		stats.(map[string]int64)["enabled"]))
}

// say streams text in short chunks, the way a model's tokens arrive.
func (a *DemoAgent) say(ctx context.Context, out Output, text string) error {
	runes := []rune(text)
	for i := 0; i < len(runes); i += 4 {
		if err := sleep(ctx, a.Step); err != nil {
			return err
		}
		out.Text(string(runes[i:min(i+4, len(runes))]))
	}
	return nil
}

func sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
