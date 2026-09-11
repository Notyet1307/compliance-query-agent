# Accord 接入分析与提案

## 已核验的仓库事实

2026-09-10 读取了 Accord Issues、README、src/profile-runtime.ts 和 docs/specs/r003-live-driver.md。树基线为 `2668ee62f249d462930bd980c8178ce5f24f7f6e`；这不是对整个仓库/CI 的完整审计。

R003 Driver 的规格仍保留：单进程、一个冻结合成 Case、四个固定 Profile、精确人工审批和单一发布者。现有运行依次涉及 RESEARCHER、ANALYST、REVIEWER、WRITER，Researcher evidence 仍通过既有合成 source manifest 的确定性接受路径。

src/profile-runtime.ts 的 InvocationBoundOutputContract（当前 GenericProfile 限 REVIEWER/WRITER）绑定 invocationId/contextDigest/profile/version/schema；不能拿该泛型函数名就推断任意新角色已经可插拔。

因此本包**没有**“注册一个 JSON 立即可用”的生产接口。role-proposal.json 是本仓库自己的提案格式，不能直接导入 Accord，更不能新增假 `/api/roles/register`。

## 推荐的业务定位

用户可见：**合规资料研究员**。能力：security.compliance.query。业务角色不与底层四个认知 Profile 混为一谈。未来可作为 RESEARCHER 阶段的领域能力，也可在正式可扩展 Profile 机制建立后绑定独立角色；应选择符合当时已批准 Spec 的最小接缝，而不是现在改枚举绕门禁。

## 请求到结果

Accord 创建/授权/冻结 Invocation → 薄适配器最小化导出 question/topic/date/scope/关联摘要 → agent-compose Run → 本智能体执行 → 验证 cqa.result/v1 → 核对身份/摘要/运行引用/状态 → 候选 Board 投影。

| 本智能体结果 | 建议候选投影（待 Accord 接缝批准） |
|---|---|
| citations | 候选 EvidenceRef；须经过 Accord 正式来源验证，不能直接 Accepted |
| claims | 候选 Claim/Observation；保持来源关联与人工复核标记 |
| INSUFFICIENT_EVIDENCE / NEEDS_REVIEW | Question/未解决项；不假称工作流完成 |
| 执行错误或 UNKNOWN | 回到持久 Invocation/Attempt 的失败/未知处理；不自动再调用 |

Source ID 不是 Accord BoardEntry ID，不能原样塞进 sourceRefs；适配器须通过权威入口建立正确映射。caseId/invocationId/contextDigest 在本服务中仅检验形状和回传，不验证它们实际存在或授予权限。

## 双方最小改动边界

本仓库负责稳定输入输出、业务限制、运行材料和数据服务契约。Accord 仓库负责受权调用入口、结果的来源资格/角色输出契约、稳定身份和状态推进、旧上下文拒绝与发布约束。

R003 的既有合成场景必须保留作回归，不能以直接替换 source manifest/模型 prompt 的方式让真实法律资料偷偷变成已接受证据。需要一个明确的新里程碑或可控试验 Spec。

## 真实联调的验收重点

同一个 Invocation 的重放/取消/UNKNOWN；不同上下文的错绑拒绝；source hash 与原文对上；陈旧结果不会发布；人工决策仍属于用户；agent-compose completion 不等于 Accord Case completion；模型成功不等于来源被接受；本地 token 不等于领域授权。

本次未读取或操作真实 Accord 数据库/Case，未创建新的 Issue/PR，未运行双方联调。S3 是候选跨仓库任务，需分别获得相关权限。
