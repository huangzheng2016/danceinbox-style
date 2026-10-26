package api

import (
	"context"
	"strconv"
	"time"

	runsvc "MODULE_NAME/biz/service/run"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/sse"
)

// writeEvents sends the replayed events, then live ones, until the run is done,
// the client goes away, or the subscription is dropped (the client then
// reconnects with Last-Event-ID). Events at or below the last sent sequence are
// skipped: the replay and the live feed may overlap.
func writeEvents(ctx context.Context, c *app.RequestContext, replay []runsvc.Event, live <-chan runsvc.Event) {
	// tells reverse proxies not to buffer the stream
	c.Response.Header.Set("X-Accel-Buffering", "no")
	stream := sse.NewStream(c)
	var last int64
	send := func(e runsvc.Event) bool {
		if e.Seq <= last {
			return true
		}
		last = e.Seq
		return stream.Publish(&sse.Event{ID: strconv.FormatInt(e.Seq, 10), Event: e.Type, Data: e.Data}) == nil
	}
	for _, e := range replay {
		if !send(e) {
			return
		}
	}
	if live == nil {
		return
	}
	// A write is the only way to notice a client that disappeared without closing.
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case e, ok := <-live:
			if !ok || !send(e) || e.Type == "done" {
				return
			}
		case <-heartbeat.C:
			if stream.Publish(&sse.Event{Event: "ping", Data: []byte("{}")}) != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
