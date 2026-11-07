namespace go api.settings

// Each field is read by a service; see biz/service/settings.Settings.
struct AppSettings {
    1: string announcement
    2: i64 run_timeout_seconds
    3: bool runs_paused
    // 0 means no limit
    4: i64 max_attachments
}

struct SettingsSnapshot {
    // 0 means never saved: settings are the defaults, updated_* are empty
    1: i64 version
    2: AppSettings settings
    // the version this one restored, 0 for an ordinary save
    3: i64 restored_from
    4: string updated_by_name
    5: i64 updated_at
}

struct SettingsResp {
    1: i32 code
    2: string msg
    3: SettingsSnapshot item
}

struct UpdateSettingsReq {
    // the version the form was loaded from; a newer saved version means a conflict (409)
    1: i64 base_version
    2: AppSettings settings
    3: i64 restored_from
}

struct ListSettingsVersionsReq {
    1: i32 page (api.query="page")
    2: i32 page_size (api.query="page_size")
}

struct ListSettingsVersionsResp {
    1: i32 code
    2: string msg
    3: list<SettingsSnapshot> items
    4: i64 total
}

struct GetSettingsVersionReq {
    1: i64 version (api.query="version")
}

struct SettingsVersionResp {
    1: i32 code
    2: string msg
    3: SettingsSnapshot item
    // the version before it (the defaults for version 1), for the diff
    4: AppSettings previous
}

struct PublicSettingsResp {
    1: i32 code
    2: string msg
    3: string announcement
}
