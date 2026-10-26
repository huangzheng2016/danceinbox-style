package user

import (
	"context"
	"errors"
	"strings"

	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"
	"MODULE_NAME/biz/errs"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct{ q *query.Query }

func NewService(q *query.Query) *Service { return &Service{q: q} }

// Login accepts a username or an email. Unknown user and wrong password produce
// the same error so the endpoint cannot be used to discover accounts.
func (s *Service) Login(ctx context.Context, identifier, password string) (*model.User, error) {
	u := s.q.User
	cond := u.Username.Eq(identifier)
	if strings.Contains(identifier, "@") {
		cond = u.Email.Eq(identifier)
	}
	user, err := u.WithContext(ctx).Where(cond).First()
	bad := errs.Unauthorized("用户名或密码错误")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bad
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil || !user.Active {
		return nil, bad
	}
	return user, nil
}

// Active returns the user if it exists and is enabled. Auth middleware calls it
// on every request, so disabling a user or changing a role takes effect at once.
func (s *Service) Active(ctx context.Context, id int64) (*model.User, error) {
	u := s.q.User
	user, err := u.WithContext(ctx).Where(u.ID.Eq(id), u.Active.Is(true)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.Unauthorized("账号不存在或已停用")
	}
	return user, err
}

// Search finds active users whose username contains q, for pickers; at most 20.
func (s *Service) Search(ctx context.Context, q string) ([]*model.User, error) {
	u := s.q.User
	do := u.WithContext(ctx).Where(u.Active.Is(true))
	if q = strings.TrimSpace(q); q != "" {
		do = do.Where(db.Contains(u.Username, q))
	}
	return do.Order(u.Username).Limit(20).Find()
}

// Names maps user ids to usernames, for showing owners next to other records.
// Unknown ids are left out.
func (s *Service) Names(ctx context.Context, ids []int64) (map[int64]string, error) {
	names := map[int64]string{}
	if len(ids) == 0 {
		return names, nil
	}
	u := s.q.User
	rows, err := u.WithContext(ctx).Select(u.ID, u.Username).Where(u.ID.In(ids...)).Find()
	for _, r := range rows {
		names[r.ID] = r.Username
	}
	return names, err
}

func (s *Service) List(ctx context.Context) ([]*model.User, error) {
	u := s.q.User
	return u.WithContext(ctx).Order(u.ID).Limit(500).Find()
}

func (s *Service) Create(ctx context.Context, username, password, role string) (*model.User, error) {
	switch role {
	case model.RoleUser, model.RoleModerator, model.RoleAdmin:
	default:
		return nil, errs.Invalid("角色无效")
	}
	if len(password) < 8 {
		return nil, errs.Invalid("密码至少 8 位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{Username: username, PasswordHash: string(hash), Role: role, Active: true}
	err = s.q.User.WithContext(ctx).Create(user)
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil, errs.Conflict("用户名已存在")
	}
	return user, err
}
