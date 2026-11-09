# danceinbox-style

一套 Hertz (Go) + Vue 3 全栈开发规范，同时是一份**可以直接运行的脚手架**：复制、改名后即可构建、启动、登录，并管理一个完整的示例资源。新功能照着示例功能和参考页面复制，所以同一个项目里每个接口、每个页面的结构和交互都一致。

入口是 [`SKILL.md`](SKILL.md)，可作为 Claude Code 的 skill 使用（放到 `~/.claude/skills/danceinbox-style/`），也可以当模板手动复制。

## 特点

- **能跑的代码，而不是文档**：脚手架每次修改都会从零生成项目，构建、测试并在浏览器里走一遍，确认可用。
- **后端**：Thrift IDL + hz 生成接口，GORM 类型化查询，Wire 依赖注入；Cookie JWT 登录、三级角色、默认全部接口需登录。
- **前端**：不用组件库，原生控件加一套共享组件；统一的青色强调色，完整的浅色和暗色主题。
- **面向运营的中文界面**：文案规则、常见错误对照和自动检查脚本，避免把字段名、枚举码和技术说明直接摆上界面。
- **两种部署规格**：完整版（MySQL + Redis）和轻量版（SQLite + 进程内缓存，单个二进制）。

## 包含什么

### 后端（`backend/scaffold/`）

| 能力 | 说明 |
|---|---|
| 接口与错误 | Thrift IDL 先行；统一错误模型和响应格式，中文错误信息直接展示给用户 |
| 认证与权限 | access 1 小时 + refresh 30 天的双 cookie；公开路由显式登记；USER / MODERATOR / ADMIN 三级角色 |
| 示例功能 | 列表、详情、新建、编辑、启停、删除、批量操作、导入、统计；状态约束，删除可重试 |
| AI 流式运行 | SSE 输出，事件带序号落库，断线按 `Last-Event-ID` 续传；支持思考折叠、工具调用、子任务时间线、停止、并发上限、重启后收尾；含 Eino ADK 与 Kimi CLI / KAP 沙箱容器扩展规范（直接复用 `github.com/huangzheng2016/klient` 客户端与事件映射，详见 `backend/agent.md`） |
| 桌面端（可选） | Wails v3 + Web 双模同源架构、单一 `Service.Call` IPC 总线、三种流通道与桌面安全守卫（`backend/wails3.md`） |
| 文件上传 | 流式落盘并计算 SHA-256，随机存储名，扩展名白名单，下载强制为附件 |
| 审计日志 | 每次变更成功后记录操作人、对象和修改前后的值；批量和导入只记一条摘要 |
| 运行时设置 | 设置按版本保存，保存时检测冲突，可对比、可恢复；各服务每次使用时读取，保存即生效 |
| 测试与发布 | `go test ./...` 不依赖外部服务；`build.sh`、`release.sh`、Dockerfile、Compose、Caddy、GitHub Actions |
| 轻量版 | `backend/scaffold-lite/`：SQLite + 进程内缓存 |

### 前端（`frontend/`）

**共享组件**（`base/`）

| 类别 | 组件 |
|---|---|
| 结构 | 应用壳（面包屑、用户菜单、按角色隐藏的导航、待办角标）、对话框、确认框、浮层、操作菜单、标签页、可拖动分栏 |
| 输入 | 开关、分段选择、下拉选择 `Select`、可搜索 / 远程 / 多选 `Combobox`、时间范围、带单位的时长和容量 `UnitInput` |
| 展示 | 状态标签、提示条、全局悬停提示 `v-tip`、版本差异 `DiffView`、时间线 `Timeline`、悬浮卡片、代码和 JSON 块、Markdown、图表、进度条、空状态、分页、排序表头 |
| 编辑 | Markdown 编辑 / 预览、代码编辑器、JSON 字段 |
| AI 与任务 | 运行记录、思考折叠、执行过程聚合折叠、工具调用卡片（含子任务时间线）、流式代码块闭合、文件上传、步骤条（扩展规范见 `frontend/agent-ui.md`） |

**组合函数**：`useList`（筛选、排序、分页、选中状态全部进 URL）、`useFetch`、`usePolling`、`useUpload`、`useWizard`、`useRouteQuery`、`useSelection`、`useDirtyGuard`、`useFormValidation`、`readEventStream` 等。

**参考页面**：列表 + 详情 + 批量、完整详情页（运行 / 附件 / 记录标签页）、批量导入向导、系统设置页、审计日志、表单对话框、带图表和公告的概览、登录、404。

**显示辅助**：`format.ts` 统一带单位的格式化和「不限」；`labels.ts` 统一枚举文案；`backLink.ts` 让跨页跳转后的返回回到来源页。

**其他**：401 自动刷新会话、会话失效后跳转登录并回跳、亮 / 暗主题首屏无闪烁、窄屏单栏、键盘可用。

**质量检查**：`pnpm check` 依次运行 Prettier、ESLint、`scripts/lint-ui.mjs`、Vitest、vue-tsc 和构建。其中 `lint-ui` 检查未定义的 class、模板里的 `JSON.stringify`、数值后拼接单位、`中文（english）` 括号尾巴、中文半角标点、原生 `<select>` 和 `title` 提示；在 `base/`、`styles/` 之外还拦截写死的颜色、原始 `z-index`、页面改写公共 class、`toLocale*`/`toFixed` 格式化和本地复制的 `labels.ts` 映射。

## 目录

```
SKILL.md                   入口：新项目步骤、核心规则、技术栈
backend/
  scaffold/                项目根目录模板：后端、IDL、配置、构建发布、CI、项目 CLAUDE.md
  scaffold-lite/           轻量版的替换文件
  workflow.md              目录结构、新增功能流程、接口约定、错误、认证、审计、运行时设置、测试、版本钉选
  data.md                  查询、写入、事务、删除、迁移、缓存规则
  deploy.md                本地运行、发布、首次部署、轻量版、CI
  agent.md                 专项能力：Eino ADK 编排、4 层中间件、会话压缩、HITL 恢复、Subagent、Trace 存储与 Kimi CLI/KAP 沙箱（github.com/huangzheng2016/klient）
  wails3.md                专项能力：Wails v3 桌面端与 Web 双模架构、IPC 通道、安全守卫、窗口持久化与绑定门禁
frontend/
  scaffold/                项目 frontend/ 模板：配置、API、路由、参考页面、测试、lint-ui
  base/                    共享组件和组合函数，原样复制到 src/base/
  styles/                  tokens.css + base.css
  assets/fonts/            自托管字体及许可证
  design-system.md         页面模式、控件选择、交互规则、文案、做减法、Token、检查清单
  agent-ui.md              专项能力：流式渲染、rAF 批处理、思考折叠、工具与过程聚合折叠、子任务时间线、HITL 卡片
scripts/verify.sh          按 SKILL.md 的命令生成项目，构建、测试并启动
scripts/e2e/               浏览器端到端用例
```

## 维护

修改后运行：

```bash
scripts/verify.sh lite         # 完整流程，包括启动并访问 /healthz
scripts/verify.sh full         # 构建与测试（不启动 MySQL 和 Redis）
E2E=1 scripts/verify.sh lite   # 另外用 Chromium 跑端到端用例
```

需要 Go、pnpm（corepack）、python3、curl。

## 许可证

- 仓库主体代码和文档：MIT，见 `LICENSE`。明确标注的第三方内容仍按各自许可证使用。
- 图标：Lucide 使用 ISC，部分图标源自 Feather（MIT）。完整文本在 `frontend/base/LUCIDE-LICENSE.txt`，复制 `base/` 时一并保留。
- 字体：Plus Jakarta Sans 与 Geist Mono 使用 SIL Open Font License 1.1，许可证随字体放在 `frontend/assets/fonts/`。
- Go 和 npm 依赖保留各自的许可证；安装、构建和分发时应保留相应声明。
