# OMP 接管交接说明

日期：2026-09-10；启动包：0.1.0-starter；设计状态：需用户与 OMP 核验。

## 已实现与未实现

README 的能力矩阵描述当前代码；实际验证以 `evidence/bootstrap/verification.json` 为准。代码有真实 CLI/API、本地知识筛选、来源完整性校验、文件回执、可配置 LLM/Connect 客户端，不是空模块。

默认样本全部 synthetic，生成是原文提取。真实 LLM、真实 OctoBus 服务、原生 capset 适配、agent-compose parser/镜像/运行、Accord 适配、目标 Go/用户机器/客户环境仍待验证。没有真实合规知识或正确率结论。

## 接管必须先核验

1. 当前目录、Git 状态/HEAD、生效指令和用户已有改动，不清理工作树。
2. 用正常、无依据、核验过期/版本冲突的输入解释结果，并对应到代码和测试，而非仅复述文档。
3. OMP、Matt Skills 的真实版本/路径/依赖；复用 `docs/agents/`，不机械重新初始化或假设 Skill 可用。
4. 工具链与实际测试；目标 Go 的可用性/安全版本，Python、Docker、daemon 与 guest 的架构及权限。
5. Accord 的最新 HEAD/Issue/批准 Spec；本次调查基线是 `2668ee6...`，README 里的旧开发指针不能代替实时 Issue。不得直接覆盖 R003。
6. 外部知识服务的真实 contract、capset 路由/权限、模型协议、数据传输与费用许可。
7. `.local`、runtime 日志、真实语料与秘密是否有泄漏/误提交风险。

## 接管回执

写 `docs/handoff-receipt.md`，记录理解、真实模块/模拟/提案、命令/环境/输出、冲突与未决项、seed 保留/调整建议、下一项任务及权限。这个文件本次不预先创建空模板，更不写入“已通过”。

第一轮允许在审批后产生检查所需的临时文件/缓存/合成回执和接管报告，不修改业务代码、Spec 验收预期、不发布工单、不 commit/push/merge/release。

## 重要冲突预警

- `integrations/accord/role-proposal.json` 是本项目提案，不是 Accord 已有导入格式。
- `protocol/compliance.proto` 是拟建领域知识服务，不是 OctoBus 自带服务。
- `deploy/agent-compose.native-octobus.proposal.yml` 用了上游原生配置字段，但本程序尚未实现其 guest 注入协议适配，不能直接视为 live 集成。
- `configs/live.example.json` 的直接 Connect 客户端与原生代理路线是两种接入路径，不要重复挂载同一能力或误称凭据始终留在 daemon。
- Go 的目标版本与生成环境实际版本不同，必须重新验证。

## 未决事项

详见 `docs/decisions-and-assumptions.md`。优先消除真正影响业务或外部副作用的歧义，不把每个小实现细节再交给用户选择。提出推荐和代价，不以默认标签/已有文档推导授权。

## 推荐推进顺序

B0 → X1 → S1 → S2 → S3 → R0。X1 不扩成“开发通用运行时”；S3 需要独立的 Accord 侧批准任务。先完整交付第一条真实价值链路，再扩知识覆盖或抽取通用组件。
