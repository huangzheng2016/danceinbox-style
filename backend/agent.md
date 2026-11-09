# Agent Architecture, Storage, and Sandboxes

Optional backend patterns for projects that go beyond the scaffold's basic `run.Agent` seam (`backend/scaffold/biz/service/run/`): multi-step Eino ADK agents, rolling session compaction, HITL interrupt/resume checkpoints, subagent trees, and isolated sandboxes (from lightweight Docker/filesystem jails to full Kimi CLI / KAP container runners with S3 workspace archives).

Read this file when building or extending an AI agent, tool execution pipeline, or sandbox runner.

## Choosing the tier

| Scenario | Recommended tier | Reference pattern |
|---|---|---|
| Single-turn or short tool-calling task inside the app | **Tier 1 — Built-in `run.Agent`** (`biz/service/run`) | Stream `out.Think`, `out.Text`, `out.Tool`; events land in `run_events` and stream over SSE |
| Multi-turn conversational agent, custom Go tools, MCP, HITL confirmation, subagents | **Tier 2 — Eino ADK (`adk.ChatModelAgent` + `adk.Runner`)** | In-process Eino runner with middleware stack, rolling session compaction, and `CheckPointStore` |
| Untrusted code execution, repo-level coding agent, long autonomous tasks | **Tier 3 — Container / Kimi Sandbox Runner** | Either Eino `filesystem.Middleware` + Docker exec, or isolated K8s Job / Docker container running Kimi CLI / KAP via `github.com/huangzheng2016/klient` |

Keep the external contract with the frontend identical across all tiers: append-only sequenced events (`seq`), replay-then-follow SSE (or cursor-paged trace API), and explicit terminal states (`succeeded`, `failed`, `canceled`, `interrupted`).

---

## 1. Eino ADK agent assembly

### Standard middleware stack

An Eino `ChatModelAgent` in production needs two layers of context protection:
1. **In-run middlewares** (bound a single multi-step ReAct turn when a tool output explodes mid-run).
2. **Before-model pair guard** (repairs broken `assistant(tool_calls) -> tool(result)` pairings before every model call; a broken pair causes deterministic upstream HTTP 400 errors and retry storms).

```go
package agent

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/patchtoolcalls"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	defaultMaxIterations  = 128    // 16 is too low for real coding/ops tasks; 128 bounds runaway loops
	defaultContextWindow  = 262144 // 256K tokens
	clearThresholdRatio   = 0.75   // reduction clears old tool results at 75% window
	compactThresholdRatio = 0.90   // in-run summarization triggers at 90% window
)

func BuildADKAgent(
	ctx context.Context,
	name, description, instruction string,
	chatModel einomodel.ChatModel,
	tools []tool.BaseTool,
	maxIterations, contextWindow int,
	returnDirectly map[string]bool,
	extra ...adk.ChatModelAgentMiddleware,
) (*adk.ChatModelAgent, error) {
	if contextWindow <= 0 {
		contextWindow = defaultContextWindow
	}
	if maxIterations <= 0 {
		maxIterations = defaultMaxIterations
	}

	patchMw, err := patchtoolcalls.New(ctx, nil)
	if err != nil {
		return nil, err
	}
	reduceMw, err := reduction.New(ctx, &reduction.Config{
		SkipTruncation: true,
		TokenCounter: func(_ context.Context, msgs []adk.Message, tls []*schema.ToolInfo) (int64, error) {
			return countTokens(msgs, tls), nil
		},
		MaxTokensForClear:         int64(float64(contextWindow) * clearThresholdRatio),
		ClearRetentionSuffixLimit: 2,
	})
	if err != nil {
		return nil, err
	}
	summaryMw, err := summarization.New(ctx, &summarization.Config{
		Model: chatModel,
		TokenCounter: func(_ context.Context, in *summarization.TokenCounterInput) (int, error) {
			return int(countTokens(in.Messages, in.Tools)), nil
		},
		Trigger: &summarization.TriggerCondition{
			ContextTokens: int(float64(contextWindow) * compactThresholdRatio),
		},
	})
	if err != nil {
		return nil, err
	}

	handlers := append([]adk.ChatModelAgentMiddleware{
		patchMw,
		reduceMw,
		summaryMw,
		newPairGuardMiddleware(), // must ride last before the model
	}, extra...)

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          name,
		Description:   description,
		Instruction:   instruction,
		Model:         chatModel,
		MaxIterations: maxIterations,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               tools,
				ExecuteSequentially: true, // set true when tools share stateful resources (e.g. terminal PTY)
			},
			ReturnDirectly:     returnDirectly, // e.g. map[string]bool{"end_turn": true, "exit_plan_mode": true}
			EmitInternalEvents: true,           // required so sub-agent events reach the outer stream
		},
		Handlers: handlers,
	})
}

func estimateTokens(s string) int64 {
	var ascii, nonASCII int
	for _, r := range s {
		if r < 128 {
			ascii++
		} else {
			nonASCII++
		}
	}
	return int64(float64(ascii)/4.0 + float64(nonASCII)/1.5)
}

func countTokens(msgs []adk.Message, tools []*schema.ToolInfo) int64 {
	var n int64
	for _, m := range msgs {
		if m == nil {
			continue
		}
		n += estimateTokens(m.Content) + estimateTokens(m.ReasoningContent)
		for _, tc := range m.ToolCalls {
			n += estimateTokens(tc.Function.Name) + estimateTokens(tc.Function.Arguments)
		}
	}
	for _, t := range tools {
		raw, _ := json.Marshal(t)
		n += estimateTokens(string(raw))
	}
	return n
}
```

### Tool-call pair guard (`pairGuardMiddleware`)

Why it is mandatory: `reduction`, `summarization`, interrupted runs, or history truncation can leave an `Assistant` message with `ToolCalls` whose `Tool` result was dropped (or an orphan `Tool` message without its call). OpenAI/Anthropic/Kimi APIs reject that sequence with HTTP 400 (`tool_call_id ... is not found`).

```go
type pairGuardMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func newPairGuardMiddleware() adk.ChatModelAgentMiddleware {
	return &pairGuardMiddleware{BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{}}
}

func (m *pairGuardMiddleware) BeforeModelRewriteState(
	ctx context.Context,
	state *adk.ChatModelAgentState,
	_ *adk.ModelContext,
) (context.Context, *adk.ChatModelAgentState, error) {
	cp := *state
	cp.Messages = sanitizeHistory(state.Messages)
	return ctx, &cp, nil
}
```

`sanitizeHistory(msgs)`:
- Drops any `schema.Tool` message whose `ToolCallID` does not match an open call from the preceding `schema.Assistant` message.
- Strips unanswered `ToolCalls` from an `Assistant` message (and drops the `Assistant` message entirely if both `Content` and remaining `ToolCalls` are empty).
- Collapses interleaved non-`Tool` messages that landed between an `Assistant` tool-call frame and its `Tool` results so the call and its results stay adjacent.

### Typed tool registration (`utils.InferTool`)

Define each tool's input struct with `json` and `jsonschema` tags and build `tool.InvokableTool` with `utils.InferTool`:

```go
type QueryEventsArgs struct {
	Query string `json:"query" jsonschema:"required,description=Search keyword"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=Max rows (1-100)"`
}

invokable, err := utils.InferTool("query_events", "Search recent events by keyword.",
	func(ctx context.Context, in QueryEventsArgs) (ToolOutput, error) {
		// Return structured business errors inside ToolOutput{OK: false, Error: "..."}
		// instead of returning a non-nil Go error, because an unhandled Go error from
		// InvokableRun halts the Eino graph instead of letting the model recover.
		return execQueryEvents(ctx, in)
	},
)
```

- **Key rule**: Normal tool execution failures (bad arguments from the model, non-zero command exit code, permission refusal) return `(ToolOutput{OK: false, Text: "..."}, nil)` so the model sees the failure as a `tool_result` and can adjust. Reserve a non-nil Go `error` for unrecoverable infrastructure failures or `compose.Interrupt`.
- **Terminal tools (`ReturnDirectly`)**: Tools like `end_turn` or `exit_plan_mode` must be listed in `ToolsConfig.ReturnDirectly: map[string]bool{"end_turn": true}`. Without `ReturnDirectly`, the model receives the tool's output and keeps looping until `MaxIterations`.

### Mid-run user steering (`BeforeChatModel`)

To let users send supplemental instructions ("steer") while a run is already in progress:

```go
Middlewares: []adk.AgentMiddleware{{
	BeforeChatModel: func(_ context.Context, state *adk.ChatModelAgentState) error {
		for _, msg := range steerQueue.Drain() {
			state.Messages = append(state.Messages, schema.UserMessage(msg))
			emit(Event{Type: "steered", Text: msg})
		}
		emit(Event{Type: "status", Phase: "thinking"})
		return nil
	},
}}
```

---

## 2. Session history & rolling compaction

Eino's `summarization` middleware only rewrites the **in-run** message state; it does not shrink the persisted multi-turn session history that is reloaded from the database at the start of the next turn. Without session-layer compaction, prompt tokens grow linearly across turns.

Rules for session-layer rolling compaction:
1. **Trigger & keep budgets**: Trigger when persisted session history exceeds `triggerTokens` (e.g. `24,000` estimated tokens); keep the newest `keepTokens` (e.g. `8,000` tokens) verbatim.
2. **Cut only on a turn boundary (`schema.User`)**: Walk backwards from the end accumulating token counts; set the split index at the earliest `schema.User` message that fits within `keepTokens`. Never split an `Assistant` tool call from its `Tool` result.
3. **Rolling summary message**: Render `msgs[:split]` into a role-labeled transcript (capping each message at ~1,200 runes so one huge tool output cannot blow up the summarizer), call `model.Generate` on a detached context with a 90s timeout, and replace `msgs[:split]` with a single `schema.UserMessage("[历史摘要] 以下是更早对话的滚动摘要...\n\n" + summary)`. Because any prior summary is part of `msgs[:split]`, facts roll forward across generations.
4. **Prompt-injection guard in the summary prompt**: Explicitly instruct the summarizer: *"工具、模型、沙箱和权限是否可用由运行时重新注入，不把历史中的能力判断或拒绝推断总结为当前能力事实；保留真实完成的操作和用户目标即可。"*
5. **Fallback**: If the summarizer call fails (e.g. gateway timeout), fall back to deterministic head-trimming at a `schema.User` boundary + `sanitizeHistory`, so the next turn remains bounded.

---

## 3. HITL interrupts & checkpoints

When a tool requires user confirmation (e.g. writing to a production host, executing a destructive SQL query, or `ask_user`), pause the Eino graph with `compose.Interrupt` and resume from `adk.CheckPointStore`:

1. **Register payload types with `gob`**: Eino serializes checkpoint state with `encoding/gob`. Every custom struct stored in interrupt state must be registered in `init()`:
   ```go
   func init() {
       gob.Register(Interaction{})
       gob.Register(InteractionState{})
   }
   ```
2. **CheckPointStore**: Implement `adk.CheckPointStore` (`Get(ctx, id) ([]byte, bool, error)` and `Set(ctx, id, data []byte) error`) backed by a `checkpoints(checkpoint_id PRIMARY KEY, data BLOB, updated_at BIGINT)` table. Use `runID` as `checkpointID`.
3. **Consuming the interrupt**:
   - In the event loop, check `if event.Action != nil && event.Action.Interrupted != nil`.
   - Extract the `InterruptCtx`, generate a one-time `confirmationNonce` (and `requestId` / `attempt`), persist the run status as `interrupted`, and emit `confirm_required` / `question_required` to the stream.
   - Park the run goroutine without emitting `done`.
4. **Resuming**:
   - Verify the caller's `ownerID` and `confirmationNonce` (constant-time compare).
   - Call `runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{Targets: map[string]any{interruptID: decisionPayload}})` and continue consuming the returned `AsyncIterator` into the same run's event stream.

---

## 4. Subagents: two proven models

### Model A: In-process Eino `AgentTool`

Mount a child `adk.Agent` as a tool on the root agent via `adk.NewAgentTool(ctx, childAgent)`.
- Set `EmitInternalEvents: true` on the root agent's `ToolsConfig`.
- Eino stamps every event with `ev.RunPath` (`len == 1` is root, `len == 2` is direct child subagent, `len >= 3` is grandchild).
- **Critical history rule**: Only append `len(ev.RunPath) <= 1` messages to the root session's persisted `[]*schema.Message`. If depth-2 subagent messages leak into root history between the root `AgentTool` call and its result, the next turn fails with an upstream 400 (`tool_call_id not found`).
- **Event attribution (`ToolCallID`)**: Track the root agent's open tool calls by tool name (`openCalls[toolName] = append(stack, callID)`). When a depth-2 event arrives from `ev.AgentName`, stamp `env.ToolCallID = latestOpenCall(ev.AgentName)`. The transcript API buckets subagent events by `subagent_events[tool_call_id]` so the UI renders them nested inside the parent tool card.

### Model B: Independent child runs (`parent_run_id`) or managed subagent pool

- **Async / heavy duty runs**: The root router dispatches a ticket; a background worker starts a separate top-level run with `ParentRunID: routerRunID`. The UI lists `childRuns` on the parent run and only opens an SSE connection to `childRunId` when the user expands that subagent timeline.
- **Interactive subagent tool**: The `subagent` tool spawns a child `Runner` and forwards live events (`subagentDelta`, `subagentToolCall`, `subagentToolResult`, `subagentDone`) tagged with `parentCallId`, `subagentId`, and `depth`.

---

## 5. Run & Event storage patterns

### Pattern 1: Atomic sequenced `run_events` (`danceinbox-style`)

Store every event before publishing to live subscribers so `Last-Event-ID` / `after_seq` never misses or duplicates an event.
- When multiple goroutines can write to the same run's log (e.g. the stream consumer + out-of-band approval audit events), allocate `seq` atomically in the database:
  ```sql
  INSERT INTO run_events(run_id, seq, type, data_json, ts)
  SELECT ?, s.next, ?, json_set(?, '$.seq', s.next), ?
  FROM (SELECT COALESCE(MAX(seq), 0) + 1 AS next FROM run_events WHERE run_id = ?) s
  RETURNING seq
  ```
- **Startup reconciliation**: On service startup, any run left in `queued` or `running` from a dead process is transitioned to `failed` (`"服务重启，运行中断"`). When `Subscribe` sees a run with `EndedAt != 0` whose stored events lack a final `done` event, synthesize a trailing `done` event so reconnecting clients settle cleanly.

### Pattern 2: Batched journal stream

When a model streams tokens at high frequency (100+ chunks/s), synchronous per-token DB inserts can stall the live stream. `WithRunJournal` solves this without losing durability:
- **Batchable events** (`delta` / `text`, `reasoning` / `think`, `toolArgs`): buffered in memory and flushed to the DB in a single batch every `150ms` or `32 KiB`, then delivered to the live stream.
- **Non-batchable & structural events** (`status`, `toolCall`, `toolResult`, `confirmRequired`, `done`, `error`, `canceled`): immediately flush all buffered chunks first, then persist the structural event synchronously before delivering it.
- **Terminal atomic guard**: Finishing a run uses compare-and-set on expected status (`WHERE id = ? AND status = ?`) together with the terminal event append, preventing cancel/finish races.

### Pattern 3: High-volume Trace storage with server-side compaction & LRU cache

When sandbox runners stream thousands of fine-grained KAP/CLI trace events (`think`, `text`, `tool_call`, `tool_result`, `step_begin`, `turn_end`, `usage`, `subagent`, `end`):
1. **Idempotent batch ingestion (`RunnerEvents`)**:
   - The container streamer buffers trace events and POSTs `application/x-ndjson` batches (up to 256 events / 16 MiB, flushing every `200ms` or 32 events with exponential backoff; events > 16 MiB are replaced with a truncated envelope carrying `sha256` and `original_bytes`).
   - The server deduplicates by unique key `(run_id, seq)` (with `epoch`): identical retries are ignored; conflicting payloads for the same `(run_id, seq)` return `409 Conflict`.
2. **Server-side trace compaction (`compactTraceRows`)**:
   - `TracePage(Compact: true)` merges consecutive `text` or `think` rows that share the same `(run_id, epoch, turn_id)` and have contiguous `seq` into a single row before sending to the browser, cutting payload size and DOM work by 10x+.
   - Completed runs (`Terminal(run.Status)`) cache their compacted `TracePage` in an in-memory byte-bounded LRU cache (`16 MiB` / 128 pages).
3. **ClickHouse archival**:
   - Long-term trace events archive to ClickHouse (`sandbox.agent_trace_events`) and query with `argMax(payload, ingested_at) ... GROUP BY seq ORDER BY seq ASC LIMIT 500` for deduplicated cursor pagination.

---

## 6. Sandbox architectures

### Architecture A: Lightweight Eino `filesystem.Middleware` + Docker jail

Best when the agent runs in-process (Go) and needs safe file manipulation plus shell execution (`python3`, `node`, `go`, `curl`, `rg`, `jq`):

1. **Three-directory jail layout**:
   - `/workspace` (read-write): per-tenant/per-group working directory on host volume.
   - `/downloads` (read-only): uploaded or cached media/attachments.
   - `/outbox` (read-write): artifacts the agent produces to send back to the user (enforcing a size cap, e.g. 30 MB).
2. **Mounting into Eino**:
   - Implement Eino's `filesystem.Backend` over the jailed paths (rejecting any `..` traversal or symlink escape outside the three roots) and `filesystem.Shell` over either a local rootless Docker container (`--read-only --network bridge -v <ws>:/workspace:rw`) or a remote `sandboxd` HTTP daemon.
   - Pass `&filesystem.MiddlewareConfig{Backend: backend, Shell: shell, CustomSystemPrompt: &prompt}` to `filesystem.New(ctx, mc)` and append the middleware to `BuildADKAgent`. This automatically injects `read_file`, `write_file`, `edit_file`, `ls`, `glob`, `grep`, and `execute`.

### Architecture B: Isolated Container Runner with Kimi CLI / KAP (`github.com/huangzheng2016/klient`)

Best when running autonomous coding/investigation agents (such as Kimi Code CLI) with custom skills, MCP servers, and persistent multi-turn workspaces across K8s Jobs or Docker containers:

1. **Data model (`Sandbox` / `Session` / `Run` / `Snapshot` / `Input`)**:
   - `Sandbox`: `{id, owner, name, runtime, image, cpu_milli, memory_mib, snapshot_id, archive_key, active_run_id, status}`. Only one active run per sandbox (`active_run_id` compare-and-set under row lock).
   - `Session`: `{id, sandbox_id, owner, profile_id, cli_session_id, archive_key}`. Tracks multi-turn conversation state across runs.
   - `Run`: `{id, owner, request_key, request_hash, sandbox_id, session_id, profile_id, status, bundle_cipher, lease_owner, lease_until, epoch, tokens_in, tokens_out, archive_key, cli_session_id}`.
   - **Idempotent `SendSession` / `CreateRun`**: Every request carries a `request_key`. The server hashes the request payload into `request_hash`; a retry with the same `(owner, request_key)` and matching `request_hash` returns the existing `Run` (or `Input`), while a different payload returns `409 Conflict`.
   - **Active-run auto-steer**: In `SendSession`, if the sandbox already has an `ActiveRunID` belonging to the same `SessionID`, the new message is queued as a `pending` row in `inputs` instead of failing with a conflict! The container's `steerPoller` polls `/internal/runs/:id/steer` every 2s, calls `PostPrompt` + `SteerPrompts` into the live Kimi turn, and ACKs via `/internal/runs/:id/steer/ack`.

2. **Sealed execution bundle (`BundleCipher`)**:
   - At run creation, resolve the `Runtime`, `Profile` (model, `max_context_size`, `capabilities`, `default_effort`, `disabled_tools`, `system_prompt`), selected `Skills` (text + base64 binary files), and `MCPConnections` (with tool allowlists and decrypted secret headers).
   - Seal the resolved `Bundle` with AEAD (`s.Keys.Seal`) into `run.BundleCipher`.
   - Issue a short-lived, run-scoped `TaskToken` (`auth.Claims{RunID: run.ID}`) passed to the container. The container never receives raw upstream LLM API keys or database credentials:
     - Model calls go to `{{ .GwURL }}/v1` using `TaskToken`, and the container sandbox service proxies them to the upstream LLM while metering token usage and enforcing model cooldowns on 429/5xx.
     - MCP calls go to `{{ .GwURL }}/internal/runs/:run_id/mcp/:conn_id` using `TaskToken`, and the container sandbox service enforces the per-run MCP tool allowlist and injects upstream headers.

3. **Kimi CLI / KAP integration via `github.com/huangzheng2016/klient`**:
   - **Never re-implement KAP REST/WebSocket clients or event mappers.** Directly import and use the standalone library `github.com/huangzheng2016/klient` (always resolve the latest version):
     ```bash
     go get github.com/huangzheng2016/klient@latest
     ```
   - Inside `runner-entrypoint`:
     - Renders `$HOME/.kimi-code/config.toml` (pointing provider `sandbox-gateway` at `GwURL/v1` with `api_key = TaskToken`, setting `default_permission_mode = "yolo"` inside the isolated container, and disabling interactive-only tools like `AskUserQuestion`).
     - Renders `$HOME/.kimi-code/SYSTEM.md` and `$HOME/.kimi-code/mcp.json`, and extracts the skills zip into `$HOME/.kimi-code/skills/`.
     - Starts `kimi web --no-open --host 127.0.0.1 --port <port> --dangerous-bypass-auth` locally inside the container and drives it through `klient`:
       - `client := klient.New(baseURL, token)` — creates the KAP client.
       - `client.Healthz(ctx)` — polls `/api/v1/healthz` until `kimi web` is ready.
       - `client.CreateSession(ctx, cwd)` — creates a new workspace session (`POST /api/v1/sessions`) when `ResumeSessionID` is empty.
       - `snap, err := client.GetSnapshot(ctx, sessionID)` -> `snap.Cursor()` (`klient.Cursor{Seq, Epoch}`) — fetches session state, `in_flight_turn`, and the durable resumption cursor (`GET /api/v1/sessions/:id/snapshot`).
       - `client.ListTasks(ctx, sessionID)` & `client.ListMCPServers(ctx, sessionID)` — inspects foreground/background tasks (`GET /api/v1/sessions/:id/tasks`) and active MCP servers/tool counts (`GET /api/v1/mcp/servers`).
       - `sub, err := client.Subscribe(ctx, sessionID, cursor)` — opens `/api/v1/ws`, performs the `client_hello` -> `server_hello` -> `subscribe` handshake with cursor resumption, automatically replies to `ping` frames with `pong`, and surfaces `resync_required` (`klient.TypeResyncRequired`), `klient.TypeReadError`, and `klient.TypeProtocolError` on `sub.Events`.
       - `receipt, err := client.PostPrompt(ctx, sessionID, klient.PromptRequest{Text, Model, PermissionMode})` — submits a user prompt (`POST /api/v1/sessions/:id/prompts`) and returns `receipt.PromptID`.
       - `client.SteerPrompts(ctx, sessionID, promptIDs)` — steers pending prompts into an active turn (`POST /api/v1/sessions/:id/prompts/:pid:steer`).
       - `client.Abort(ctx, sessionID)` — aborts an active session turn (`POST /api/v1/sessions/:id:abort`).
       - `mapper := &klient.KAPMapper{MainPromptID: promptID, SkipFirstUsage: isResume}` — normalizes raw KAP events via `mapper.Map(ev)` -> `(kind, payload, ok)` into `"think"`, `"text"`, `"tool_call"`, `"tool_result"`, `"step_begin"`, `"turn_end"`, `"usage"`, `"error"`, and `"end"` (converting cumulative KAP token usage snapshots into per-step deltas and skipping the initial historical usage snapshot when `SkipFirstUsage: isResume` is true).

   ```go
   client := klient.New(fmt.Sprintf("http://127.0.0.1:%d", port), "")
   if err := waitUntilHealthy(ctx, client); err != nil {
       return err
   }

   sessionID := bundle.ResumeSessionID
   isResume := sessionID != ""
   if !isResume {
       sess, err := client.CreateSession(ctx, workDir)
       if err != nil {
           return err
       }
       sessionID = sess.ID
   }

   snap, err := client.GetSnapshot(ctx, sessionID)
   if err != nil {
       return err
   }
   _, _ = client.ListMCPServers(ctx, sessionID)
   _, _ = client.ListTasks(ctx, sessionID)

   sub, err := client.Subscribe(ctx, sessionID, snap.Cursor())
   if err != nil {
       return err
   }
   defer sub.Close()

   receipt, err := client.PostPrompt(ctx, sessionID, klient.PromptRequest{
       Text:           bundle.Prompt,
       Model:          bundle.Model,
       PermissionMode: "yolo",
   })
   if err != nil {
       return err
   }

   mapper := &klient.KAPMapper{
       MainPromptID:   receipt.PromptID,
       SkipFirstUsage: isResume,
   }
   for ev := range sub.Events {
       if ev.Type == klient.TypeResyncRequired {
           // Re-fetch GetSnapshot(ctx, sessionID) and re-subscribe from snap.Cursor()
           break
       }
       if kind, payload, ok := mapper.Map(ev); ok {
           streamer.Enqueue(kind, ev.Seq, ev.Epoch, payload)
           if kind == "end" {
               break
           }
       }
   }
   ```

4. **S3 workspace archive & snapshot continuity (`RunnerArchive`)**:
   - Never mount host directories or the Docker socket into runner containers.
   - Before a run starts, if the session or sandbox has an `ArchiveKey` (or initial `SnapshotID`), `runner-entrypoint` downloads and extracts the `.tar.gz` archive (`work/`, `kimi-home/`, `runner-logs/`).
   - When the run finishes, `runner-entrypoint` tars and gzips `work/`, `kimi-home/`, and `runner-logs/` and uploads the stream to `/internal/runs/:id/archive`.
   - The server streams to a temp file while computing SHA-256 and enforcing `MaxSnapshotBytes`, validates the tar structure (`validateSessionArchive`: rejects `..`, absolute paths, symlinks/device nodes, duplicate entries, >100,000 files, or decompression bombs > `4 * MaxSnapshotBytes`), and stores it in S3 under `sessions/<run_id>/<sha256>.tar.gz`. The next turn in that session downloads this archive and passes `ResumeSessionID: session.CLISessionID`, giving full filesystem and conversation continuity across ephemeral containers.
