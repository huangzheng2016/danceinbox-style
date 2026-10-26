namespace go api.auth

struct User {
    1: i64 id (go.tag='json:"id,string"')
    2: string username
    3: string email
    4: string role
}

struct LoginReq {
    1: string identifier (api.body="identifier", api.vd="len($) > 0")
    2: string password (api.body="password", api.vd="len($) > 0")
}

struct UserResp {
    1: i32 code
    2: string msg
    3: User user
}

struct SearchUsersReq {
    1: string q (api.query="q")
}

struct UserListResp {
    1: i32 code
    2: string msg
    3: list<User> items
}
