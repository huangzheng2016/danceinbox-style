package run

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"MODULE_NAME/biz/dal/db/dbtest"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"
	"MODULE_NAME/biz/errs"
)

// scriptAgent emits two text chunks, one tool call, then waits for release or cancellation.
type scriptAgent struct {
	release chan struct{}
	fail    error
}

func (a *scriptAgent) Run(ctx context.Context, ex *model.Example, out Output) error {
	out.Text("你")
	out.Text("好")
	_, _ = out.Tool(ctx, "lookup", map[string]int{"n": 1}, func(context.Context) (any, error) { return "ok", nil })
	select {
	case <-a.release:
		return a.fail
	case <-ctx.Done():
		return ctx.Err()
	}
}

// unlockedCache satisfies cache.Cache for tests; every lock is free.
type unlockedCache struct{}

func (unlockedCache) Get(context.Context, string, any) error                { return errors.New("miss") }
func (unlockedCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (unlockedCache) Delete(context.Context, ...string) error               { return nil }
func (unlockedCache) Ping(context.Context) error                            { return nil }
func (unlockedCache) TryLock(context.Context, string, time.Duration) (func(), bool, error) {
	return func() {}, true, nil
}

func setup(t *testing.T, agent Agent) (*Service, *query.Query, *model.Example) {
	t.Helper()
	q := dbtest.New(t)
	ex := &model.Example{Name: "a", Status: model.StatusEnabled}
	if err := q.Example.WithContext(context.Background()).Create(ex); err != nil {
		t.Fatal(err)
	}
	s, err := NewService(q, unlockedCache{}, agent)
	if err != nil {
		t.Fatal(err)
	}
	return s, q, ex
}

func collect(t *testing.T, replay []Event, live <-chan Event) []Event {
	t.Helper()
	out := append([]Event{}, replay...)
	if live == nil {
		return out
	}
	timeout := time.After(5 * time.Second)
	for {
		select {
		case e, ok := <-live:
			if !ok {
				return out
			}
			if len(out) == 0 || e.Seq > out[len(out)-1].Seq {
				out = append(out, e)
			}
			if e.Type == "done" {
				return out
			}
		case <-timeout:
			t.Fatal("stream did not finish")
		}
	}
}

func types(events []Event) []string {
	var ts []string
	for _, e := range events {
		ts = append(ts, e.Type)
	}
	return ts
}

func TestStreamReplaysThenFollows(t *testing.T) {
	ctx := context.Background()
	agent := &scriptAgent{release: make(chan struct{})}
	s, _, ex := setup(t, agent)
	run, err := s.Start(ctx, ex.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Subscribe mid-run: part of the output is already stored.
	time.Sleep(100 * time.Millisecond)
	replay, live, stop, err := s.Subscribe(ctx, run.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	close(agent.release)
	events := collect(t, replay, live)

	want := []string{"status", "text", "text", "tool_call", "tool_result", "status", "done"}
	if got := types(events); len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, e := range events {
		if e.Seq != int64(i+1) {
			t.Fatalf("event %d has seq %d: gap or duplicate", i, e.Seq)
		}
	}
	var done map[string]string
	_ = json.Unmarshal(events[len(events)-1].Data, &done)
	if done["status"] != model.RunSucceeded {
		t.Fatalf("done status %q", done["status"])
	}

	// Reconnecting after seq 3 resumes exactly from 4, from the store.
	replay, live, _, _ = s.Subscribe(ctx, run.ID, 3)
	if live != nil || len(replay) != 4 || replay[0].Seq != 4 {
		t.Fatalf("resume: %d events from seq %d", len(replay), replay[0].Seq)
	}
}

func TestCancelAndFailureAreRecorded(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		act    func(s *Service, runID int64, a *scriptAgent)
		status string
		err    string
	}{
		{"stop", func(s *Service, id int64, _ *scriptAgent) { _ = s.Cancel(ctx, id) }, model.RunCanceled, "已停止"},
		{"agent error", func(_ *Service, _ int64, a *scriptAgent) { a.fail = errors.New("模型不可用"); close(a.release) }, model.RunFailed, "模型不可用"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agent := &scriptAgent{release: make(chan struct{})}
			s, q, ex := setup(t, agent)
			run, _ := s.Start(ctx, ex.ID, 1)
			replay, live, stop, _ := s.Subscribe(ctx, run.ID, 0)
			defer stop()
			time.Sleep(50 * time.Millisecond)
			tc.act(s, run.ID, agent)
			collect(t, replay, live)
			got, _ := q.Run.WithContext(ctx).Where(q.Run.ID.Eq(run.ID)).First()
			if got.Status != tc.status || got.Error != tc.err || got.EndedAt == 0 {
				t.Fatalf("got %s %q ended %d", got.Status, got.Error, got.EndedAt)
			}
		})
	}
}

func TestOneActiveRunPerExample(t *testing.T) {
	ctx := context.Background()
	agent := &scriptAgent{release: make(chan struct{})}
	s, _, ex := setup(t, agent)
	if _, err := s.Start(ctx, ex.ID, 1); err != nil {
		t.Fatal(err)
	}
	var e *errs.Error
	if _, err := s.Start(ctx, ex.ID, 1); !errors.As(err, &e) || e.Code != errs.CodeConflict {
		t.Fatalf("second start: %v", err)
	}
	close(agent.release)
}

func TestRestartFailsInterruptedRuns(t *testing.T) {
	ctx := context.Background()
	s, q, ex := setup(t, &scriptAgent{release: make(chan struct{})})
	stale := &model.Run{ExampleID: ex.ID, Status: model.RunRunning}
	_ = q.Run.WithContext(ctx).Create(stale)

	if _, err := NewService(q, unlockedCache{}, s.agent); err != nil {
		t.Fatal(err)
	}
	replay, live, _, _ := s.Subscribe(ctx, stale.ID, 0)
	if live != nil || len(replay) != 1 || replay[0].Type != "done" {
		t.Fatalf("an interrupted run must still end its stream: %v", types(replay))
	}
}
