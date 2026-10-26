package run

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"MODULE_NAME/biz/dal/db/model"
)

// Agent produces a run's output. Replace DemoAgent with the project's model
// call: stream tokens through Text, wrap each tool invocation in Tool.
type Agent interface {
	Run(ctx context.Context, ex *model.Example, out Output) error
}

type Output interface {
	// Text appends a chunk of the Markdown answer.
	Text(delta string)
	// Tool records a tool call, runs fn, and records its result or error.
	Tool(ctx context.Context, name string, args any, fn func(context.Context) (any, error)) (any, error)
}

type emitter struct {
	s     *Service
	runID int64

	mu    sync.Mutex
	seq   int64
	tools int
}

func (e *emitter) Text(delta string) {
	e.emit("text", map[string]string{"delta": delta})
}

func (e *emitter) Tool(ctx context.Context, name string, args any, fn func(context.Context) (any, error)) (any, error) {
	e.mu.Lock()
	e.tools++
	id := fmt.Sprintf("t%d", e.tools)
	e.mu.Unlock()
	e.emit("tool_call", map[string]any{"id": id, "name": name, "args": args})
	start := time.Now()
	out, err := fn(ctx)
	result := map[string]any{"id": id, "duration_ms": time.Since(start).Milliseconds()}
	if err != nil {
		result["error"] = err.Error()
	} else {
		result["output"] = out
	}
	e.emit("tool_result", result)
	return out, err
}

func (e *emitter) status(status, msg string) {
	r := e.s.q.Run
	_, err := r.WithContext(context.Background()).Where(r.ID.Eq(e.runID)).UpdateSimple(r.Status.Value(status), r.Error.Value(msg), r.EndedAt.Value(endedAt(status)))
	if err != nil {
		log.Printf("run %d: update status: %v", e.runID, err)
	}
	e.emit("status", map[string]string{"status": status, "error": msg})
}

func endedAt(status string) int64 {
	if status == model.RunRunning || status == model.RunQueued {
		return 0
	}
	return time.Now().Unix()
}

// emit stores the event, then pushes it to live subscribers. Storage uses a
// detached context: a canceled run still records how it ended.
func (e *emitter) emit(typ string, data any) {
	raw, _ := json.Marshal(data)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seq++
	ev := &model.RunEvent{RunID: e.runID, Seq: e.seq, Type: typ, Data: string(raw)}
	if err := e.s.q.RunEvent.WithContext(context.Background()).Create(ev); err != nil {
		log.Printf("run %d: store event %d: %v", e.runID, e.seq, err)
	}
	e.s.publish(e.runID, Event{Seq: e.seq, Type: typ, Data: raw})
}
