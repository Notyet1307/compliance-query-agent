# Compliance Query Agent｜安全合规查询智能体

**版本：0.1.0-starter。状态：供用户与 OMP 核验的启动包，不是生产合规产品。**

目标：把等保、关基保护、安全政策、行业知识、商用密码应用安全性评估的**资料查询与证据化回答**封装为独立智能体，由 agent-compose 承载，通过 OctoBus 取得经批准的数据，未来作为 Accord 的受管角色参与 Case。

本仓库不复制 Accord 的任务、黑板、审批、发布或调度系统。用户已建立公开仓库；许可证、后续 Issue 和发布仍须单独授权。

## 从哪里读

使用者先读 [`START_HERE.md`](START_HERE.md)。OMP 按 [`AGENTS.md`](AGENTS.md) → [`docs/handoff.md`](docs/handoff.md) → 产品/Spec/架构/代码的顺序接管。

## 当前真正能运行什么

| 能力 | 本次状态 |
|---|---|
| Go CLI 输入 JSON、离线检索、日期/范围筛选、返回引用 | 已实现；样本全部为虚构测试数据 |
| 五个查询领域、无依据/过期核验/版本重叠/未生效处理 | 已实现并有自动化检查；不是五个真实知识库 |
| 结果回执持久化、同输入重放、同 ID 不同输入拒绝 | 已实现；绑定本地数据目录，不能跨无共享卷的沙箱保证去重 |
| 带令牌的本地 HTTP API | 已实现；仅允许 loopback，不是公网服务或多租户服务 |
| LLM chat-completions 客户端、结果引用校验 | S1 单文档真实调用已获用户接受；S2 固定 `baizhi-chat/grok-4.6` 经受管代理生成测试草稿，保留 provider Token 用量及未知状态 |
| OctoBus Connect unary 客户端 | 已实现；仅模拟协议验证，实际知识服务/凭据 BLOCKED |
| agent-compose 声明、guest 镜像构建材料 | 已在固定 ARM64 镜像上完成 parser/build/command 运行；限 X1 合成试验 |
| agent-compose 原生 OctoBus capset 路径 | X1 候选已实现并真实调用合成 Search；固定 proto/grpcurl，不是通用发现或生产接口 |
| S2 受管模型、稳定身份与持久回执 | 当前真实查询／草稿切片已获用户接受；权限拒绝、无依据零生成、正常草稿和原生 run 归属实测通过。跨容器／新 sandbox 重放延期，整 S2 未完成；见 [公开验收摘要](evidence/s2-query/accepted-summary.json) |
| Accord 角色接入 | 接口提案/候选工单；未修改 Accord，也未接入 |
| 真实法规、标准、政策全文 | 未内置；需来源、时效与授权核验 |

## 本地运行

工具链目标见 `.go-version`；生成环境 Go 1.23.2 的历史证据仍在 `evidence/bootstrap/verification.json`。本机已使用 Go 1.27.1 darwin/arm64 验证，并构建 linux/arm64 guest；见 [接管回执](docs/handoff-receipt.md) 与 [X1 执行记录](docs/specs/x1.md)。`go.mod` 的 `1.23.0` 是语言兼容下限，不是生产版本推荐。

```bash
go version
make verify
make demo
```

不依赖第三方 Go 模块，不需要数据库、模型 Key、OctoBus 或 Docker 就能执行离线演示。`extractive` 模式是真实的引用片段提取，**不是模拟 LLM，也不冒充语义理解**。

```bash
go run ./cmd/compliance-agent query --config configs/demo.json --input examples/mlps.json
go run ./cmd/compliance-agent query --config configs/demo.json --input examples/insufficient.json
go run ./cmd/compliance-agent query --config configs/demo.json --input examples/conflict.json
```

所有示例固定查询日期 `2026-09-10`，以便重复验证；不是自动查询“今天”。修改问题、日期或配置后使用新的 `requestId`；同 ID 不同输入返回 `IDEMPOTENCY_CONFLICT`。

## 真实接入前必须理解

`configs/live.example.json` 默认 `networkApproved:false`，知识端点仍为占位，不能直接作为已验收配置运行。固定百智云 `grok-4.6` 已用于获单独授权的 S1／S2 测试；S2 真实链路绑定指定平台定制候选，不证明未修补上游或任意部署可用。当前只记录 Token 用量，不实现 10 元／20 次闸门；缺失用量不记零，`max_tokens:2048` 不是完整计费上限。每次新真实运行仍须另行授权并确认账号、地域／留存、费用和外发资料。接口与限制见 [开发说明](docs/development.md#s2-当前真实查询草稿验收)。

Accord 当前核验基线为 `2668ee62f249d462930bd980c8178ce5f24f7f6e`。该版本 R003 仍限定四个固定 Profile 和合成案例，不提供本启动包可直接使用的动态“注册合规角色”接口。详见 `integrations/accord/README.md`，不能声称填一个角色 JSON 即已集成。

## 交给 OMP

复制 [`prompts/01_OMP接管合规查询智能体.prompt.md`](prompts/01_OMP接管合规查询智能体.prompt.md)。第一轮仅接管核验和设计补缺，输出 `docs/handoff-receipt.md`，不自动进入开发，不发布 Issue，不提交或推送。

产品行为以版本化 Spec 为准，正式任务状态以 [GitHub Issues](https://github.com/Notyet1307/compliance-query-agent/issues) 为准；[种子目录](docs/issue-seeds/README.md) 仅保留候选与发布映射。用户已接受当前 S2 真实查询／测试草稿并授权本次 commit、push、工单回写和 merge；不授予新模型／资源操作、release、生产、真实资料或 Accord 接入权限。
