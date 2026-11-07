namespace go api.audit

struct AuditLog {
    1: i64 id (go.tag='json:"id,string"')
    // "0" for changes made without a session; the page shows 系统
    2: i64 actor_id (go.tag='json:"actor_id,string"')
    3: string actor_name
    // model.Audit* constants; labels in the frontend's AUDIT_ACTION
    4: string action
    5: string target_type
    // "0" when the change has no single target (batch, import)
    6: i64 target_id (go.tag='json:"target_id,string"')
    7: string target_name
    8: string summary
    // JSON {"before": {...}, "after": {...}}, or "" when there is nothing to compare
    9: string detail
    10: i64 created_at
}

struct ListAuditLogsReq {
    1: i32 page (api.query="page")
    2: i32 page_size (api.query="page_size")
    3: i64 actor_id (api.query="actor_id")
    4: string action (api.query="action")
    5: string target_type (api.query="target_type")
    6: i64 target_id (api.query="target_id")
    // created_at within [created_from, created_to), unix seconds; 0 means unbounded
    7: i64 created_from (api.query="created_from")
    8: i64 created_to (api.query="created_to")
}

struct ListAuditLogsResp {
    1: i32 code
    2: string msg
    3: list<AuditLog> items
    4: i64 total
}

struct ExampleHistoryReq {
    1: i64 id (api.query="id")
    2: i32 page (api.query="page")
    3: i32 page_size (api.query="page_size")
}
