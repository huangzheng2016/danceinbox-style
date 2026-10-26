namespace go api.attachment

struct Attachment {
    1: i64 id (go.tag='json:"id,string"')
    2: i64 example_id (go.tag='json:"example_id,string"')
    3: string name
    4: i64 size
    5: string content_type
    6: string sha256
    7: i64 created_at
}

// multipart/form-data: example_id plus one "file" part
struct UploadAttachmentReq {
    1: i64 example_id (api.form="example_id")
}

struct AttachmentResp {
    1: i32 code
    2: string msg
    3: Attachment item
}

struct ListAttachmentsReq {
    1: i64 example_id (api.query="example_id")
}

struct ListAttachmentsResp {
    1: i32 code
    2: string msg
    3: list<Attachment> items
}

struct AttachmentIDReq {
    1: i64 id (go.tag='json:"id,string"')
}

struct DownloadAttachmentReq {
    1: i64 id (api.query="id")
}
