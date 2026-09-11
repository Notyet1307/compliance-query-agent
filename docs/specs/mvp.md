# CQA-MVP/r0：安全合规查询启动规格

状态：DRAFT_FOR_OWNER_REVIEW。该 Spec 描述当前 M0 可运行骨架及推荐 M1 目标，二者不混淆。尚未发布正式任务，不构成真实模型、数据传输或跨仓库实现授权。

## A. 已确认的产品结果

覆盖用户提出的五个领域；智能体独立封装；包含自己的业务逻辑与 LLM 接口；可通过 OctoBus 获取外部数据；由 agent-compose 承载；未来通过 Accord 的受管角色使用。

## B. M0 当前可运行行为

### 输入

`cqa.query/v1`。必填 `requestId, question, topic, asOfDate, jurisdiction, industry`。主题严格限定五个枚举；地域当前仅 `CN`。`asOfDate` 必须为有效日期；不把“今天”在后台转换成不同结果。`industry` 作为范围过滤，不是权限证明。

问题不超过 8000 UTF-8 字节；HTTP/CLI 输入不超过 65536 字节；拒绝未知字段、重复 JSON key 和多个 JSON 文档。

可选 Accord 关联 `caseId, invocationId, contextDigest` 必须一起出现；digest 为 32 字节的十六进制字符串。该校验仅验证形状和回传绑定，不查询 Accord，也不验证真实主体或权限。

### 资料资格

资料必须通过 schema/字段/日期/text hash 校验。只选相同 topic/jurisdiction、行业匹配或 `all` 的片段。当前 relevance 是管理员关键词对子串的匹配；最多选择五个片段。

按查询日满足 `publishedAt <= asOfDate`、`effectiveFrom <= asOfDate < effectiveTo`；结束日期为空时无已记录结束日。历史查询可使用结束有效期前的已废止版本；未生效/已过期片段不作为当前依据。这里约束的是“截至该日可知且适用的资料”，尚不支持以后公布的追溯生效规则。

相关来源未经审核、属于草案/未知状态或核验日期不足时，返回 `NEEDS_REVIEW`，不调用模型。相同 documentId/locator 的有效版本重叠且内容/版本不同也阻断。跨文件语义冲突识别不在 M0。

### 输出

`cqa.result/v1`，始终包含输入/配置/资料摘要、数据模式、生成模式、查询日期、claims/citations、原因码、限制和生成时间。

| status | 意义 |
|---|---|
| REFERENCE_ONLY | 只有原文提取，不代表 LLM 回答或合规通过 |
| DRAFT_READY | 已得到带引用的 LLM 参考草稿，需人工复核 |
| INSUFFICIENT_EVIDENCE | 没有足够可用证据或模型返回依据不足 |
| NEEDS_REVIEW | 来源时效/状态/重叠版本需要人工处理 |

业务不足/待复核也可以 HTTP 200，因为请求已得到真实业务结果；执行失败返回非 2xx 和固定错误码。不得用 HTTP 200 或 `healthz up` 证明业务依据正确。

所有源码默认输出 `humanReviewRequired:true`、`entailmentVerified:false`。citation 的 URL、文档、版本、原文和 hash 由检索结果产生，不让模型生成或覆盖。LLM claims 只能引用本次集合内的 ID，但这不构成语义蕴含证明。

### 重复与错误

同 requestId + 相同输入摘要/配置摘要 → 已完成则返回完全相同保存结果，不重新检索或调用模型。同 ID 不同输入/配置 → 409。配置摘要包括本地 corpus snapshot 摘要和 Agent 版本；模型凭据只绑定引用，不把 token hash 放进公开响应。

持久化预留后崩溃、模型失败、响应格式错误、未知结果 → 后续同 ID 返回 `PREVIOUS_EXECUTION_UNRESOLVED`，不静默重试。存储不可用则停止，不绕开记录继续联网。未提供自动恢复/清理/重试命令。

### 安全

默认 local/extractive、禁止网络、允许明确标记 synthetic 的演示样本。live 配置拒绝 synthetic。联网须部署配置批准；只允许明确配置的端点，HTTPS 或显式 loopback HTTP；不跟随重定向，不读取代理环境，不让请求更改工具或模型。API 仅 loopback + 至少 32 字符 token。

OctoBus 客户端只调用 `Search` 的 Connect unary 形状，不访问 admin API。该服务名是本项目提案，OctoBus 上游不自带这个知识服务。

## C. M0 可重复验收

| AC | 行为 | 对应检查 |
|---|---|---|
| A01 | 五类样例能返回带标记的参考片段 | TestFiveDomainsAndExceptions |
| A02 | 无依据不调用 LLM/不造条文 | TestNoEvidenceDoesNotCallLLM |
| A03 | 时效不足、版本重叠、未来/历史边界正确 | 五域异常、TestHistoricalInterval |
| A04 | text digest 被改即拒绝 | TestDigestTampering |
| A05 | live 不接受 synthetic | TestSyntheticDenied |
| A06 | 引用伪造、无引用被拒绝 | TestModelCitationValidation / ForgedCitation |
| A07 | 同请求重放不重复调用、不同输入拒绝 | Idempotency / ConcurrentReservation / Restart |
| A08 | 未完成预留不自动重试 | ReservedNeverRetries / DeadlineAndReceipt |
| A09 | 未批准网络、非 TLS、重定向被拒绝 | NetworkApprovalAndTLS / NoRedirect |
| A10 | API 鉴权、大小、loopback 限制生效 | ServerAuthAndLoopback / APIRejectsOversizedBody |
| A11 | Connect/LLM 正确构造请求、处理响应 | 本地 httptest 协议测试 |
| A12 | CLI 重启、真实 loopback HTTP 进程可运行 | scripts/smoke.py |

A11 是模拟协议契约测试，不是实际 LLM/OctoBus 兼容性认证。M0 不以部署材料或架构图代替真实集成验收。

## D. 推荐 M1 验收边界（待批准）

用户批准一批真实资料及参考问答；在支持的目标 Go、真实 agent-compose guest、只读 OctoBus capset、选定模型上运行。至少一次从 Accord 触发受管角色，返回候选证据/说明，由 Accord 验收并按自身规则完成用户可见反馈。

真实验收应覆盖错误 capset、外部服务不可达、过期版本、无依据、错误关联/旧上下文、模型超时、取消后状态不明、重复任务、沙箱重建后的回执/运行引用，以及 secret reflection/网络越权。对应实际代码与证据必须绑定提交 SHA；缺任一关键环境标 BLOCKED。

M1 不包括扩大 R003 scope 的隐式授权；S3 必须先在 Accord 自身 Spec/Issue 流程中批准接缝。
