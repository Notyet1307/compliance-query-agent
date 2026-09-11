# Compliance Query Agent｜安全合规查询智能体

**版本：0.1.0-starter。状态：供用户与 OMP 核验的启动包，不是生产合规产品。**

目标：把等保、关基保护、安全政策、行业知识、商用密码应用安全性评估的**资料查询与证据化回答**封装为独立智能体，由 agent-compose 承载，通过 OctoBus 取得经批准的数据，未来作为 Accord 的受管角色参与 Case。

本仓库不复制 Accord 的任务、黑板、审批、发布或调度系统。仓库名是建议；没有创建 GitHub 仓库、发布 Issue、选择公开/私有或许可证。

## 从哪里读

使用者先读 [`START_HERE.md`](START_HERE.md)。OMP 按 [`AGENTS.md`](AGENTS.md) → [`docs/handoff.md`](docs/handoff.md) → 产品/Spec/架构/代码的顺序接管。

## 当前真正能运行什么

| 能力 | 本次状态 |
|---|---|
| Go CLI 输入 JSON、离线检索、日期/范围筛选、返回引用 | 已实现；样本全部为虚构测试数据 |
| 五个查询领域、无依据/过期核验/版本重叠/未生效处理 | 已实现并有自动化检查；不是五个真实知识库 |
| 结果回执持久化、同输入重放、同 ID 不同输入拒绝 | 已实现；绑定本地数据目录，不能跨无共享卷的沙箱保证去重 |
| 带令牌的本地 HTTP API | 已实现；仅允许 loopback，不是公网服务或多租户服务 |
| LLM chat-completions 客户端、结果引用校验 | S1-02 本地模拟候选：固定 `baizhi-chat/grok-4.6` 接口、记录 provider Token 用量及未知状态；真实调用 NOT_RUN |
| OctoBus Connect unary 客户端 | 已实现；仅模拟协议验证，实际知识服务/凭据 BLOCKED |
| agent-compose 声明、guest 镜像构建材料 | 已提供；实际 parser/build/run BLOCKED |
| agent-compose 原生 OctoBus capset 路径 | 提案；自动发现与代理调用适配未实现 |
| Accord 角色接入 | 接口提案/候选工单；未修改 Accord，也未接入 |
| 真实法规、标准、政策全文 | 未内置；需来源、时效与授权核验 |

## 本地运行

建议开发/部署工具链见 `.go-version`；本次生成环境的实际工具链、命令和结果见 `evidence/bootstrap/verification.json`。`go.mod` 中 `1.23.0` 是语言兼容下限，不是生产版本推荐。本次可执行验证使用环境现有 Go 1.23.2；目标 Go 1.27.1 的验证尚未执行。

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

`configs/live.example.json` 默认 `networkApproved:false`；模型端点来自本机 OMP 的 `baizhi-chat/grok-4.6` 配置，不是已通过真实集成的配置，知识端点仍为占位。按用户后续修订，当前只记录 Token 用量，不再实现 10 元／20 次闸门或发送前计数证明；旧 `trialDir` 已删除。缺失用量不记零，`max_tokens:2048` 不作为完整计费上限。真实运行前仍须核验百智云模型可用性、实际处理地域、条款及公开样例范围并单独授权；不沿用百炼北京的价格或留存结论。接口依据与可运行验证见 [S1-02 开发说明](docs/development.md#s1-02-token-用量候选)。

Accord 当前核验基线为 `2668ee62f249d462930bd980c8178ce5f24f7f6e`。该版本 R003 仍限定四个固定 Profile 和合成案例，不提供本启动包可直接使用的动态“注册合规角色”接口。详见 `integrations/accord/README.md`，不能声称填一个角色 JSON 即已集成。

## 交给 OMP

复制 [`prompts/01_OMP接管合规查询智能体.prompt.md`](prompts/01_OMP接管合规查询智能体.prompt.md)。第一轮仅接管核验和设计补缺，输出 `docs/handoff-receipt.md`，不自动进入开发，不发布 Issue，不提交或推送。

产品行为以版本化 Spec 为准；工单正式发布后，以 GitHub 为唯一任务状态来源。候选工单目前均未发布、未批准执行。
