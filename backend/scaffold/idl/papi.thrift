include "base.thrift"
include "api/auth.thrift"

namespace go papi

service UserAdminService {
    auth.UserListResp ListUsers(1: base.EmptyReq req) (api.get="/papi/user/list")
}
