# danceinbox-style

一套全栈项目开发规范（AI Agent Skill）：统一的后端脚手架 + 前端设计系统，让新项目搭建快、页面风格一致。

## 概览

本仓库是一个 Skill（入口为 `SKILL.md`），一份规范两个部分：

### Part 1 - Scaffold（项目脚手架）

从零搭建 **Hertz (Go) + Vue 3** 全栈项目的脚手架规范：

- **后端**：Thrift IDL + `hz` 代码生成（IDL 先行，路由/模型/handler 全部由 IDL 契约生成）+ Wire 编译期 DI + GORM / `gorm/gen` 类型化查询层
- **认证授权**：JWT 双 Cookie（access 1h + refresh 30d，HttpOnly）认证，默认 `role` 列 + `RequireRoleMw` 的 RBAC，Casbin 为可选升级路径
- **发布流程**：`build.sh` 增量构建 + `release.sh` rsync 发布 + Docker Compose 编排 + Caddy 边缘代理（ACME 托管 TLS）
- **测试约定**：TDD 节奏、表驱动 Go 测试、`httptest` 模式、真实 MySQL（`TEST_MYSQL_DSN` 门控）
- **轻量 profile**：SQLite（纯 Go `glebarez/sqlite`）+ 进程内 `LocalRedis`，免 MySQL / 免 Redis 的单二进制部署，与完整 profile 共享同一套调用接缝（`IRedis`、gen query 层），后续可平滑升级

### Part 2 - Style（前端设计系统）

Vue 3 前端设计系统，**原生控件、无组件库**：

- `tokens.css` + `base.css` 全局样式：扁平不透明表面 + 单一青色强调色，支持 `micro` / `rounded` / `square` 三种形状和 `white` / `soft-white` / `warm-paper` 三种浅色表面，默认 `micro + soft-white`
- 暗色主题由 `:root.dark` 独立覆盖；设计 profile 只在脚手架或明确重设计时选择并记录，页面不得各自换肤
- 固定应用壳（icon 侧栏可折叠、壳渲染页头）、可拖拽分栏、中文运营文案约定
- 共享组件与 composable 模板（`Icon` / `Loading` / `Pagination` / `BaseDialog` / `SplitResizer` / `useFetch` / `errorBus` 等），整组复制、只允许改 `EDIT:` 标记处
- 请求取消、超时与竞态防护，表格错误重试，分页归一化，自托管字体及 SIL OFL 许可证
- Element Plus / Tailwind 等组件库与 CSS 框架明确禁用

## 目录结构

```
SKILL.md                  # Skill 入口：两个 Iron Rule、启动序列、Stack Lock、Red Flags
backend/                  # Part 1 后端参考文档（按需加载）
  idl-workflow.md         #   Thrift 模板、hz_update.sh、中间件挂载规则、生成物防漂移
  backend-skeleton.md     #   main.go 启动序列、统一模型注册表、Wire、配置 schema、embed 接线
  auth-jwt-casbin.md      #   JWT Cookie 方案、刷新流程、Casbin 升级、路由权限覆盖
  release-flow.md         #   build.sh / release.sh / Dockerfile / Caddyfile / compose / CI
  testing-conventions.md  #   TDD、数据库与前端共享行为测试模板
  data-conventions.md     #   gorm/gen + IRedis 使用规范（规范性）
  embed-strategy.md       #   可选：zstd 压缩 embed（dist > 50MB 时启用）
  lite-profile.md         #   轻量 profile 模板（SQLite / LocalRedis / 角色列 RBAC）
frontend/
  design-profiles.md      #   形状与浅色表面的选择、默认值、HTML 属性与升级边界
  skeleton.md             #   前端骨架：axios 刷新拦截、Pinia auth store、i18n、双入口构建、pnpm check
  tokens.css              #   完整 CSS 变量（profile + 亮色/暗色），逐字复制
  base.css                #   全局样式表，替代组件库，逐字复制
  code-editor.css         #   CodeEditor / JsonField 高亮叠层样式
  assets/fonts/           #   自托管 WOFF2 字体与 SIL OFL 许可证
  layouts/                #   应用壳 / 页面卡片 / 填充滚动与分栏布局契约
  components/             #   按钮、表单、卡片、chips、对话框、代码组件规范
  templates/              #   共享组件与 composable 模板（整组复制）
scripts/
  check-refs.sh           #   引用、profile、字体、token 与模板导入校验
```

## 使用方式

这是一个给 AI 编码助手使用的 Skill：将本仓库放入 agent 的 skill 目录，agent 在「从零搭建 Hertz+Vue 项目」或「编写该设计系统下的 Vue 页面」时会自动激活，按 `SKILL.md` 的启动序列与规范执行。

- 新项目：按 Part 1「Mandatory Boot Sequence」12 步顺序执行，选择并记录基础设施、路由和设计 profile，最终 `CLAUDE.md` / `AGENTS.md` 中固化 Universal Discipline 与这些决策
- 已有项目写页面：先读取项目已记录的设计 profile，复制 `tokens.css` / `base.css` / `templates/` 和自托管字体，按「Page Patterns」选型，用「Red Flags」清单做完成前自查

仓库内的一致性可用 `./scripts/check-refs.sh` 校验。
