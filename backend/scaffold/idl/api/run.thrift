namespace go api.run

struct Run {
    1: i64 id (go.tag='json:"id,string"')
    2: i64 example_id (go.tag='json:"example_id,string"')
    // queued | running | succeeded | failed | canceled
    3: string status
    4: string error
    5: i64 created_at
    6: i64 ended_at
}

struct StartRunReq {
    1: i64 example_id (go.tag='json:"example_id,string"')
}

struct RunIDReq {
    1: i64 run_id (go.tag='json:"run_id,string"')
}

struct GetRunReq {
    1: i64 run_id (api.query="run_id")
}

struct RunResp {
    1: i32 code
    2: string msg
    3: Run item
}

struct ListRunsReq {
    1: i64 example_id (api.query="example_id")
    2: i32 page (api.query="page")
    3: i32 page_size (api.query="page_size")
}

struct ListRunsResp {
    1: i32 code
    2: string msg
    3: list<Run> items
    4: i64 total
}

// Server-Sent Events. Resume with the Last-Event-ID header, or `after` when the
// client cannot set headers. Events: status, text, tool_call, tool_result, done.
struct StreamRunReq {
    1: i64 run_id (api.query="run_id")
    2: i64 after (api.query="after")
}
