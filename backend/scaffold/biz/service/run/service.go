// Package run executes agents in the background and streams their output.
//
// Every event a run emits is stored with a per-run sequence number before it is
// pushed to live subscribers. Clients replay from the store and then follow the
// live feed, so a dropped connection resumes with Last-Event-ID and nothing is
// lost or duplicated. The event types are the contract with the frontend's
// useAgentStream: status, text, tool_call, tool_result, done.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"
	"MODULE_NAME/biz/errs"
	"MODULE_NAME/biz/extension/cache"

	"gorm.io/gorm"
)

const (
	// runs across the whole process; more would starve the agent backend
	maxConcurrent = 4
	runTimeout    = 5 * time.Minute
)

type Event struct {
	Seq  int64
	Type string
	Data json.RawMessage
}

type Service struct {
	q     *query.Query
	cache cache.Cache
	agent Agent
	slots chan struct{}

	mu      sync.Mutex
	subs    map[int64]map[chan Event]struct{}
	cancels map[int64]context.CancelFunc
}

func NewService(q *query.Query, c cache.Cache, agent Agent) (*Service, error) {
	s := &Service{
		q: q, cache: c, agent: agent,
		slots:   make(chan struct{}, maxConcurrent),
		subs:    map[int64]map[chan Event]struct{}{},
		cancels: map[int64]context.CancelFunc{},
	}
	// A restart kills in-flight runs; record that instead of leaving them "running" forever.
	r := q.Run
	_, err := r.WithContext(context.Background()).
		Where(r.Status.In(model.RunQueued, model.RunRunning)).
		UpdateSimple(r.Status.Value(model.RunFailed), r.Error.Value("服务重启，运行中断"), r.EndedAt.Value(time.Now().Unix()))
	return s, err
}

// Start begins a run of the example and returns immediately; output arrives as events.
func (s *Service) Start(ctx context.Context, exampleID, userID int64) (*model.Run, error) {
	e := s.q.Example
	ex, err := e.WithContext(ctx).Where(e.ID.Eq(exampleID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NotFound("示例不存在")
	}
	if err != nil {
		return nil, err
	}
	if ex.Status != model.StatusEnabled {
		return nil, errs.Conflict("示例已停用，不能运行")
	}

	// The check for an active run and the insert must not interleave between requests.
	unlock, ok, err := s.cache.TryLock(ctx, fmt.Sprintf("run:start:%d", exampleID), 10*time.Second)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.Conflict("正在启动，请稍候")
	}
	defer unlock()

	r := s.q.Run
	active, err := r.WithContext(ctx).Where(r.ExampleID.Eq(exampleID), r.Status.In(model.RunQueued, model.RunRunning)).Count()
	if err != nil {
		return nil, err
	}
	if active > 0 {
		return nil, errs.Conflict("该示例已有运行中的任务")
	}
	select {
	case s.slots <- struct{}{}:
	default:
		return nil, errs.Conflict("运行中的任务过多，请稍后再试")
	}

	run := &model.Run{ExampleID: exampleID, Status: model.RunQueued, CreatedBy: userID}
	if err := r.WithContext(ctx).Create(run); err != nil {
		<-s.slots
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(context.Background(), runTimeout)
	s.mu.Lock()
	s.cancels[run.ID] = cancel
	s.mu.Unlock()
	go s.execute(runCtx, run.ID, ex)
	return run, nil
}

func (s *Service) execute(ctx context.Context, runID int64, ex *model.Example) {
	defer func() {
		<-s.slots
		s.mu.Lock()
		s.cancels[runID]()
		delete(s.cancels, runID)
		s.mu.Unlock()
	}()
	em := &emitter{s: s, runID: runID}
	em.status(model.RunRunning, "")

	err := s.agent.Run(ctx, ex, em)
	status, msg := model.RunSucceeded, ""
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		status, msg = model.RunCanceled, "已停止"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		status, msg = model.RunFailed, "运行超时"
	case err != nil:
		status, msg = model.RunFailed, err.Error()
	}
	em.status(status, msg)
	em.emit("done", map[string]string{"status": status})
}

// Cancel stops a running run. Stopping a finished run is a no-op.
func (s *Service) Cancel(ctx context.Context, runID int64) error {
	if _, err := s.Get(ctx, runID); err != nil {
		return err
	}
	s.mu.Lock()
	cancel := s.cancels[runID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (s *Service) Get(ctx context.Context, runID int64) (*model.Run, error) {
	r := s.q.Run
	run, err := r.WithContext(ctx).Where(r.ID.Eq(runID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NotFound("运行记录不存在")
	}
	return run, err
}

func (s *Service) List(ctx context.Context, exampleID int64, page, size int32) ([]*model.Run, int64, error) {
	r := s.q.Run
	offset, limit := db.Page(page, size)
	return r.WithContext(ctx).Where(r.ExampleID.Eq(exampleID)).Order(r.ID.Desc()).FindByPage(offset, limit)
}

// Subscribe returns the stored events after `after` and, unless the run has
// finished, a channel of the events that follow. Call stop when done.
func (s *Service) Subscribe(ctx context.Context, runID, after int64) (replay []Event, live <-chan Event, stop func(), err error) {
	run, err := s.Get(ctx, runID)
	if err != nil {
		return nil, nil, nil, err
	}
	// Register before reading the store: an event emitted in between is then in
	// both, and the caller drops it by sequence, instead of being in neither.
	ch := make(chan Event, 256)
	s.mu.Lock()
	if s.subs[runID] == nil {
		s.subs[runID] = map[chan Event]struct{}{}
	}
	s.subs[runID][ch] = struct{}{}
	s.mu.Unlock()
	stop = func() {
		s.mu.Lock()
		if _, ok := s.subs[runID][ch]; ok {
			delete(s.subs[runID], ch)
			close(ch)
		}
		s.mu.Unlock()
	}

	ev := s.q.RunEvent
	rows, err := ev.WithContext(ctx).Where(ev.RunID.Eq(runID), ev.Seq.Gt(after)).Order(ev.Seq).Find()
	if err != nil {
		stop()
		return nil, nil, nil, err
	}
	for _, r := range rows {
		replay = append(replay, Event{Seq: r.Seq, Type: r.Type, Data: json.RawMessage(r.Data)})
	}
	if n := len(replay); n > 0 && replay[n-1].Type == "done" {
		stop()
		return replay, nil, func() {}, nil
	}
	// Ended without a done event: the process died mid-run (see NewService).
	if run.EndedAt != 0 {
		stop()
		seq := after
		if n := len(replay); n > 0 {
			seq = replay[n-1].Seq
		}
		data, _ := json.Marshal(map[string]string{"status": run.Status})
		return append(replay, Event{Seq: seq + 1, Type: "done", Data: data}), nil, func() {}, nil
	}
	return replay, ch, stop, nil
}

func (s *Service) publish(runID int64, e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs[runID] {
		select {
		case ch <- e:
		default:
			// A subscriber this far behind is dropped; it reconnects and replays from the store.
			delete(s.subs[runID], ch)
			close(ch)
		}
	}
	if e.Type == "done" {
		for ch := range s.subs[runID] {
			close(ch)
		}
		delete(s.subs, runID)
	}
}
