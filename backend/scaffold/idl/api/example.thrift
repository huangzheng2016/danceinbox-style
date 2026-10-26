namespace go api.example

struct Example {
    1: i64 id (go.tag='json:"id,string"')
    2: string name
    3: string status
    4: string remark
    5: i64 owner_id (go.tag='json:"owner_id,string"')
    6: string owner_name
    7: i64 created_at
    8: i64 updated_at
    9: string config
}

struct ListExamplesReq {
    1: i32 page (api.query="page")
    2: i32 page_size (api.query="page_size")
    3: string q (api.query="q")
    4: string status (api.query="status")
    // "name", "-updated_at", ...; empty means newest first
    5: string sort (api.query="sort")
    // updated_at within [updated_from, updated_to), unix seconds; 0 means unbounded
    6: i64 updated_from (api.query="updated_from")
    7: i64 updated_to (api.query="updated_to")
}

struct ListExamplesResp {
    1: i32 code
    2: string msg
    3: list<Example> items
    4: i64 total
}

struct GetExampleReq {
    1: i64 id (api.query="id")
}

struct ExampleResp {
    1: i32 code
    2: string msg
    3: Example item
}

struct SaveExampleReq {
    1: i64 id (go.tag='json:"id,string"')
    2: string name (api.vd="len($) > 0 && len($) <= 64")
    3: string remark
    4: bool enabled
    5: i64 owner_id (go.tag='json:"owner_id,string"')
    6: string config
}

struct SetExampleEnabledReq {
    1: i64 id (go.tag='json:"id,string"')
    2: bool enabled
}

struct ExampleIDReq {
    1: i64 id (go.tag='json:"id,string"')
}

struct BatchExamplesReq {
    // ids as strings: the json ",string" option does not apply to lists
    1: list<string> ids (api.vd="len($) > 0 && len($) <= 100")
    2: string action (api.vd="$ == 'enable' || $ == 'disable' || $ == 'delete'")
}

struct BatchExamplesResp {
    1: i32 code
    2: string msg
    3: i32 done
    4: i32 skipped
}

struct ImportRow {
    1: string name
    2: string remark
    3: bool enabled
    4: string config
}

struct ImportExamplesReq {
    1: list<ImportRow> rows (api.vd="len($) > 0 && len($) <= 500")
}

struct ImportFailure {
    // 1-based, as the user sees it in the preview
    1: i32 row
    2: string reason
}

struct ImportExamplesResp {
    1: i32 code
    2: string msg
    3: i32 created
    4: list<ImportFailure> failed
}

struct ExampleStatsReq {
    1: i32 days (api.query="days")
}

struct DayCount {
    1: string date
    2: i32 created
}

struct ExampleStatsResp {
    1: i32 code
    2: string msg
    3: list<DayCount> days
}
