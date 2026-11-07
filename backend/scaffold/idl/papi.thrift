include "base.thrift"
include "api/auth.thrift"
include "api/audit.thrift"
include "api/settings.thrift"

namespace go papi

service UserAdminService {
    auth.UserListResp ListUsers(1: base.EmptyReq req) (api.get="/papi/user/list")
}

service SettingsAdminService {
    settings.SettingsResp GetSettings(1: base.EmptyReq req) (api.get="/papi/settings/get")
    // ADMIN only (see _updatesettingsMw)
    settings.SettingsResp UpdateSettings(1: settings.UpdateSettingsReq req) (api.post="/papi/settings/update")
    settings.ListSettingsVersionsResp ListSettingsVersions(1: settings.ListSettingsVersionsReq req) (api.get="/papi/settings/versions")
    settings.SettingsVersionResp GetSettingsVersion(1: settings.GetSettingsVersionReq req) (api.get="/papi/settings/version")
}

service AuditAdminService {
    audit.ListAuditLogsResp ListAuditLogs(1: audit.ListAuditLogsReq req) (api.get="/papi/audit/list")
}
