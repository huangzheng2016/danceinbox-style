# danceinbox-style

Hertz (Go) + Vue 3 全栈项目的 AI Agent Skill(入口 `SKILL.md`)。它不是一套规范文档,而是一份可以直接运行的脚手架:复制、改名后即可构建、启动、登录,并完整地管理一个示例资源。新功能通过复制示例功能和参考页面来开发,所以同一项目里每个接口、每个页面的结构和交互都一致。

## 包含什么

**后端** (`backend/scaffold/`)
- Thrift IDL + hz 代码生成,gorm/gen 类型化查询,Wire 依赖注入
- 统一错误模型 (`biz/errs` + `resp.Fail`),中文错误信息直接显示给用户
- Cookie JWT 会话(access 1h + refresh 30d),默认全部路由需要登录、公开路由显式登记,三级角色
- 示例功能:列表/详情/新建/编辑/启停/删除/批量/导入/统计,含状态约束与幂等删除
- AI 流式运行:SSE 走 IDL,事件落库带序号,断线按 Last-Event-ID 续传;工具调用事件;单对象单任务 + 全局并发上限;停止;进程重启后收尾
- 文件上传:流式落盘 + SHA-256,随机存储名,扩展名白名单,下载强制 attachment
- 不依赖外部服务即可运行的测试 (`go test ./...`)
- `build.sh`、`release.sh`、Dockerfile、Compose、Caddy、GitHub Actions CI
- 轻量 profile (`backend/scaffold-lite/`):SQLite + 进程内缓存的单二进制部署

**前端** (`frontend/`)
- 无组件库:`styles/` 全局样式 + `base/` 共享组件
  - 结构:应用壳(面包屑、用户菜单)、对话框、确认、浮层 (Popover)、操作菜单、标签页、分栏
  - 输入:开关、分段选择、可搜索/远程/多选下拉 (Combobox)、时间范围筛选
  - 展示:状态标签、提示条 (Alert)、悬浮卡片 (HoverCard)、可复制文本、代码/JSON 块、Markdown、图表 (ECharts)、进度条、表格状态、空状态、分页、排序表头
  - 编辑:Markdown 编辑/预览、代码编辑器、JSON 字段
  - AI 与任务:运行记录 (AgentTranscript)、工具调用卡片 (ToolCallCard)、文件上传 (FileUpload)、步骤条 (Steps)
- 组合函数:`useUpload`(进度/取消/重试)、`usePolling`(完成后再排程、隐藏暂停、退避)、`useWizard`、`readEventStream`(SSE 解析与续传)、`useList`(筛选/排序/分页/选中全部进 URL,防抖、取消、过期响应保护)、`useFetch`、`useRouteQuery`、`useSelection`(批量)、`useDirtyGuard`(未保存提醒)、`useFormValidation`、`usePageTitle`
- 参考页面:列表+详情+批量 (`ExampleList.vue`)、完整详情页 (`ExampleDetail.vue`,含运行/附件标签页)、批量导入向导 (`ExampleImport.vue`)、表单对话框、带图表的概览、登录、404
- 401 自动刷新会话、会话失效跳转登录并回跳、后端错误信息进入 Toast
- 亮/暗主题(首屏前生效,无闪烁)、窄屏单栏、键盘可用
- Vitest 行为测试 + ESLint + Prettier + vue-tsc,统一由 `pnpm check` 执行

## 目录

```
SKILL.md                   入口:新项目步骤、核心规则、技术栈
backend/
  scaffold/                项目根目录模板(后端、IDL、配置、构建发布、CI、项目 CLAUDE.md)
  scaffold-lite/           轻量 profile 的替换文件
  workflow.md              目录结构、新增功能流程、IDL/Handler/鉴权/测试约定、版本钉选
  data.md                  查询、写入、事务、删除、迁移、缓存规则
  deploy.md                本地运行、版本、发布、首次部署、轻量 profile、CI
frontend/
  scaffold/                项目 frontend/ 模板(配置、API、鉴权、路由、参考页面、测试)
  base/                    共享组件与组合函数,原样复制到 src/base/
  styles/                  tokens.css + base.css
  assets/fonts/            自托管字体及 OFL 许可证
  design-system.md         页面模式、控件选择、交互规则、文案、Token、检查清单
scripts/verify.sh          按 SKILL.md 原样生成一个项目并构建、测试、启动
scripts/e2e/               浏览器端到端用例(E2E=1 时由 verify.sh 运行)
```

## 维护

修改 skill 后运行:

```bash
scripts/verify.sh lite    # 完整流程,包括启动并访问 /healthz
scripts/verify.sh full    # 构建与测试(不启动 MySQL/Redis)
E2E=1 scripts/verify.sh lite   # 另外用 Chromium 跑完整的端到端用例
```

需要 Go、pnpm (corepack)、python3、curl。

## 许可证

- 仓库主体代码和文档:MIT,见 `LICENSE`。明确标注的第三方内容仍按各自许可证使用。
- 图标:Lucide 使用 ISC;部分 Lucide 图标源自 Feather,对应 MIT 与 Cole Bemis 版权声明。官方完整文本位于 `frontend/base/LUCIDE-LICENSE.txt`,复制 `base/` 时随组件一起保留。
- 字体:Plus Jakarta Sans 与 Geist Mono 均为 SIL Open Font License 1.1,完整许可证随字体位于 `frontend/assets/fonts/`。
- Go 和 npm 依赖保留其各自许可证;安装、构建和分发时应保留相应许可声明。
