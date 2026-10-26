package api

import (
	"context"
	"strconv"

	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/errs"
	"MODULE_NAME/biz/handler/resp"
	attachment "MODULE_NAME/biz/model/api/attachment"
	auth "MODULE_NAME/biz/model/api/auth"
	example "MODULE_NAME/biz/model/api/example"
	run "MODULE_NAME/biz/model/api/run"
	"MODULE_NAME/container"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// DAL model -> API model. Services speak model.*, handlers speak the hz types.

func toUser(u *model.User) *auth.User {
	if u == nil {
		return nil
	}
	return &auth.User{ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role}
}

func toUsers(us []*model.User) []*auth.User { return mapAll(us, toUser) }

// owners maps OwnerID to username; see ownerNames.
func toExample(e *model.Example, owners map[int64]string) *example.Example {
	return &example.Example{
		ID: e.ID, Name: e.Name, Status: e.Status, Remark: e.Remark,
		OwnerID: e.OwnerID, OwnerName: owners[e.OwnerID], Config: e.Config,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

// ownerNames loads the owners of rows with one query instead of one per row.
func ownerNames(ctx context.Context, rows ...*model.Example) (map[int64]string, error) {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if r.OwnerID != 0 {
			ids = append(ids, r.OwnerID)
		}
	}
	return container.Default.User.Names(ctx, ids)
}

func toRun(r *model.Run) *run.Run {
	return &run.Run{ID: r.ID, ExampleID: r.ExampleID, Status: r.Status, Error: r.Error, CreatedAt: r.CreatedAt, EndedAt: r.EndedAt}
}

func toAttachment(a *model.Attachment) *attachment.Attachment {
	return &attachment.Attachment{
		ID: a.ID, ExampleID: a.ExampleID, Name: a.Name, Size: a.Size,
		ContentType: a.ContentType, Sha256: a.SHA256, CreatedAt: a.CreatedAt,
	}
}

func parseIDs(raw []string) ([]int64, error) {
	ids := make([]int64, len(raw))
	for i, s := range raw {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, errs.Invalid("ID 格式错误:" + s)
		}
		ids[i] = id
	}
	return ids, nil
}

func mapAll[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func respondExample(ctx context.Context, c *app.RequestContext, row *model.Example) {
	owners, err := ownerNames(ctx, row)
	if err != nil {
		resp.Fail(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, &example.ExampleResp{Item: toExample(row, owners)})
}
