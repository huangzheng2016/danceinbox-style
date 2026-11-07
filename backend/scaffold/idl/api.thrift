include "base.thrift"
include "api/auth.thrift"
include "api/example.thrift"
include "api/run.thrift"
include "api/attachment.thrift"
include "api/audit.thrift"
include "api/settings.thrift"

namespace go api

service AuthService {
    auth.UserResp Login(1: auth.LoginReq req) (api.post="/api/auth/login")
    base.EmptyResp RefreshToken(1: base.EmptyReq req) (api.post="/api/auth/refresh")
    base.EmptyResp Logout(1: base.EmptyReq req) (api.post="/api/auth/logout")
    auth.UserResp GetMe(1: base.EmptyReq req) (api.get="/api/user/me")
    auth.UserListResp SearchUsers(1: auth.SearchUsersReq req) (api.get="/api/user/search")
}

service ExampleService {
    example.ListExamplesResp ListExamples(1: example.ListExamplesReq req) (api.get="/api/example/list")
    example.ExampleResp GetExample(1: example.GetExampleReq req) (api.get="/api/example/get")
    example.ExampleResp CreateExample(1: example.SaveExampleReq req) (api.post="/api/example/create")
    example.ExampleResp UpdateExample(1: example.SaveExampleReq req) (api.post="/api/example/update")
    base.EmptyResp SetExampleEnabled(1: example.SetExampleEnabledReq req) (api.post="/api/example/set_enabled")
    base.EmptyResp DeleteExample(1: example.ExampleIDReq req) (api.post="/api/example/delete")
    example.BatchExamplesResp BatchExamples(1: example.BatchExamplesReq req) (api.post="/api/example/batch")
    example.ImportExamplesResp ImportExamples(1: example.ImportExamplesReq req) (api.post="/api/example/import")
    example.ExampleStatsResp ExampleStats(1: example.ExampleStatsReq req) (api.get="/api/example/stats")
    // the example's audit entries, for its 记录 tab; visible to anyone who can see the example
    audit.ListAuditLogsResp ExampleHistory(1: audit.ExampleHistoryReq req) (api.get="/api/example/history")
}

service PublicSettingsService {
    // the settings every signed-in page may use (the announcement)
    settings.PublicSettingsResp PublicSettings(1: base.EmptyReq req) (api.get="/api/settings/public")
}

service RunService {
    run.RunResp StartRun(1: run.StartRunReq req) (api.post="/api/example/run/start")
    base.EmptyResp CancelRun(1: run.RunIDReq req) (api.post="/api/example/run/cancel")
    run.RunResp GetRun(1: run.GetRunReq req) (api.get="/api/example/run/get")
    run.ListRunsResp ListRuns(1: run.ListRunsReq req) (api.get="/api/example/run/list")
    // text/event-stream; streaming routes end in /stream (see Caddyfile)
    base.EmptyResp StreamRun(1: run.StreamRunReq req) (api.get="/api/example/run/stream")
}

service AttachmentService {
    attachment.AttachmentResp UploadAttachment(1: attachment.UploadAttachmentReq req) (api.post="/api/example/attachment/upload")
    attachment.ListAttachmentsResp ListAttachments(1: attachment.ListAttachmentsReq req) (api.get="/api/example/attachment/list")
    base.EmptyResp DeleteAttachment(1: attachment.AttachmentIDReq req) (api.post="/api/example/attachment/delete")
    // the file itself, not JSON
    base.EmptyResp DownloadAttachment(1: attachment.DownloadAttachmentReq req) (api.get="/api/example/attachment/download")
}
