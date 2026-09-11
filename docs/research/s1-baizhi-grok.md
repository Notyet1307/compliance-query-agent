# S1 百智云／Grok 与 Token 用量核验

核验日：2026-09-11。用户在授权 S1-02 实施后修订：后端使用 OMP 的 baizhiyun provider、`grok-4.6`，当前仅记录 Token。本文替代旧模型的当前接入依据；[旧百炼记录](s1-model-candidate.md) 保留为历史。没有真实模型调用、凭据解析、安装或账号操作；网页／配置事实不等于网关集成通过。

## 本机 OMP 的实际配置

只对指定 OMP `models.yml` 做安全字段投影，没有输出 apiKey、header 值或其他 provider 配置，没有读取 auth 数据库、执行凭据命令或运行在线模型发现。

| 字段 | 观察 |
|---|---|
| 实际 provider ID | `baizhi-chat`，不是字面名称 `baizhiyun` |
| 模型选择器 | `baizhi-chat/grok-4.6` |
| API 家族 | `openai-completions`，即 Chat Completions；不是 Responses |
| baseUrl | `https://ai-api-gateway.app.baizhi.cloud/api/openai` |
| 模型 compat | `maxTokensField:max_tokens`、`supportsReasoningEffort:false`、`supportsStore:false` |
| 模型声明 | `reasoning:true`、contextWindow 500000、maxTokens 500000；后者只是本机配置值，不是认证的最大输出或价格保证 |
| 另一 provider | 存在 `baizhi-responses`，本次投影没有其显式 grok-4.6 模型条目；没有擅自切换到它 |

OMP 内置资料 `omp://models.md` 说明 provider 命名空间、baseUrl、API 家族和 compat；`omp://provider-endpoint-constraints.md` 明确不同 OpenAI 兼容网关的参数、推理和 usage 不可互换。现有 Go 客户端只参考这一路由配置，不依赖 OMP 编码代理或自动复制其认证信息。

[INFERENCE] 按所选 Chat Completions 客户端的相对路径规则，候选完整 URL 是 baseUrl 加 `/chat/completions`，不另加 `/v1`。代码固定该 URL 并仅对 loopback 做模拟；尚无百智云独立接口文档或真实请求证明此账户／路由可用。

## 百智云第一方资料能证明什么

[百智云 API 网关产品页](https://www.baizhi.cloud/landing/api-gateway) 说明：

- “协议兼容与自动转换”：对外支持 OpenAI／Anthropic，内部可自动转换。
- “供应商与模型统一管理”：管理员管理 Base URL、API Key、上游模型映射及模型价格；FAQ 明确“供应商 Key、Base URL、上游模型名等信息由管理员统一维护，成员只需要使用被授权的 API Key 和模型名称。”
- “调用审计与成本可观测”：记录状态、模型／供应商／成员、Token、缓存、时延、错误与费用。

这些是产品能力声明，不是本实例的字段级契约。公开页面未给出此路由的精确 URL 拼接、JSON Object 参数、max_tokens 语义或 usage JSON 格式。网关基址匿名只读访问返回 401、控制台指南依赖应用页面；这不能证明模型可用性。没有访问推理、模型目录或账号接口。

## xAI 第一方模型与用量事实

[xAI Grok 4.6 模型页](https://docs.x.ai/developers/models/grok-4.6) 列出 `grok-4.6`、500,000 上下文、推理和结构化输出能力；该页列出的美国区域不证明百智云的实际路由，更不能把新模型视为已确认境内处理。

[结构化输出说明](https://docs.x.ai/developers/model-capabilities/text/structured-outputs) 列出 response_format 的 json_object／json_schema／text。[推理说明](https://docs.x.ai/developers/model-capabilities/text/reasoning) 说明 Grok 4.6 的推理及 effort 行为；但本机百智云 compat 关闭 effort 字段支持，候选不发送 reasoning_effort，也不发送 Qwen 的 enable_thinking。模型推理存在不等于可以记录其正文。

[Chat Completions 参考](https://docs.x.ai/developers/rest-api-reference/inference/chat-completions) 区分 visible `message.content` 与 `reasoning_content`，列出 prompt_tokens、completion_tokens、total_tokens 和 details。直接 xAI 对 max_completion_tokens／max_tokens 的说明不能替代百智云参数契约；本候选 max_tokens 只是输出请求参数，不证明完整计费输出上界。

[缓存用量说明](https://docs.x.ai/developers/advanced-api-usage/prompt-caching/usage-and-pricing) 将 cached_tokens 放在 prompt_tokens_details，不应再次加到输入或总量。推理与 completion 的算术关系不能仅凭字段名称推断：Chat 参考的总量描述与官方例子不一致；[Deferred Chat 示例](https://docs.x.ai/developers/advanced-api-usage/deferred-chat-completions) 明确展示 prompt=26、completion=168、reasoning=304、total=498、cached=4，并称响应与非 deferred 相同。不能强制 total=prompt+completion，也不能一律把 reasoning 再加到 total。

OMP 的 `omp://provider-quirks.md` 说明标准 usage 会归一化输入、输出、cacheRead、cacheWrite，部分 provider 还有特殊语义；`omp://provider-streaming-internals.md` 仍把计数口径和可用性留给 provider。因此本服务保存原始网关计数，不冒充 OMP 归一化账目；非流式不需要 stream_options。

## 当前最小实施决定

- 固定 baizhi-chat／grok-4.6 的单次非流式 Chat 请求；复用 Go HTTP／CLI／回执，不新增 SDK、模型发现、计费平台或自动重试。
- 记录 provider 的 prompt、completion、total 及可选 cached／reasoning 整数；只采用 provider total 聚合，不估算、不重新求和、不按旧价格换算。
- 完整、部分、未知、未调用分别记录；显式 0 不是缺失。无效 usage 整体 unknown，不能把有效草稿变成费用闸门失败。
- 拒绝输出仍保存已收到的可解析计数；超时／HTTP 错误／无法解析响应的用量未知，不假定未计费。保存失败／崩溃及回执删除造成的缺口不能由本地记录补证。
- 只消费回答 content；隐藏推理、原始 provider 错误和凭据不写公开证据。代理自带账单／审计能力不是本仓库已经对账或验证过的能力。

## 未执行与下一入口

真实 Grok 推理、网关兼容／身份、账号可用性、实际地域、留存条款与账单对账均 NOT_RUN／待核验。当前完成的是配置及公开资料研究和本地软件候选；仅记录 Token 不限制花费，也不授予真实运行权限。新 provider 的数据处理风险须重新确认；来源／许可／人工答案及冻结样例仍按 [S1/r1](../specs/s1.md) 审核。候选验收与可运行命令见 [开发说明](../development.md#s1-02-token-用量候选)。
