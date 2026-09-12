# 开发、运行和验证

## 工具链和依赖

目标 Go `1.27.1` 由 `.go-version` 声明，不会自动安装；`go.mod go 1.23.0` 是语言兼容下限。Go `1.23.2 linux/amd64` 仅是启动包的历史生成环境；本机 Go `1.27.1 darwin/arm64` 已用于 S1 的独立树验证与 CLI 运行，具体证据见下文。这不替代其他平台、guest 或生产环境验证。

Go 代码只用标准库，没有第三方模块，因此没有 `go.sum`；这不是遗漏锁文件。测试/构建均设置 `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`，不会自动下载工具链或模块。验证脚本另需 Python 3；不安装任何 Python 第三方包。

```bash
make verify   # go test、vet、build、真实 CLI/loopback HTTP smoke
make demo
make build
```

`make verify` 会创建 `.local/verification/`、临时配置和合成回执，并在测试中启动本机 loopback HTTP mock/服务进程；不会主动访问互联网。构建产物在 `bin/`。基础检查不会启动 OMP/agent-compose/OctoBus，也不会替用户安装组件。

可另执行 `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -race -count=1 ./...`；是否有 C 编译器/race 支持须按平台验证。本次已运行与未运行列表以证据文件为准。

## CLI

```bash
./bin/compliance-agent query --config configs/demo.json --input examples/mlps.json
./bin/compliance-agent query --config configs/demo.json --input examples/stale.json
```

也支持本地 stdin `--input -`，但 agent-compose 的 command REPL 不是进程 stdin 透传，不能据此声称可以直接把 JSON 管道传进 guest。运行中收到 SIGINT/SIGTERM 会取消本地请求；不保证远端模型/沙箱已停止，未知保持未知。

## 本地 API

先生成仅当前会话使用的随机 token，避免写入 shell 历史中的字面量：

```bash
export CQA_API_TOKEN="$(python3 -c 'import secrets; print(secrets.token_urlsafe(32))')"
./bin/compliance-agent serve --config configs/demo.json
```

服务端仅绑定 `127.0.0.1:8088`。在保有同一变量的受控环境发请求：

```bash
curl -sS http://127.0.0.1:8088/healthz
curl -sS http://127.0.0.1:8088/v1/query \
  -H "Authorization: Bearer ${CQA_API_TOKEN}" \
  -H 'Content-Type: application/json' \
  --data-binary @examples/mlps.json
```

不要把 token 发给他人或把带敏感数据的命令输出存进公开仓库。没有 UI、上传接口、管理员 API、结果列表接口或租户选择接口；这些不是空返回，而是未提供。

## agent-compose 本地合成演示（本次 BLOCKED）

先按上游安装方式取得真实可用的 agent-compose CLI/daemon/driver 和对应 guest 镜像，不在本包中自动安装。选定的 guest 镜像用 digest 固定；`Dockerfile.guest` 保留上游 ENTRYPOINT/CMD，只加入 Go 二进制与合成材料。

```bash
# 先在受控环境准备实际镜像引用，不能原样保留占位符。
# Linux amd64 guest：
make guest-binary GOARCH=amd64
# ARM64 guest 则改 GOARCH=arm64；不得运行错架构二进制。
docker build -f deploy/Dockerfile.guest \
  --build-arg BASE_GUEST_IMAGE='<已批准的上游guest镜像@sha256:digest>' \
  -t compliance-query-agent-guest:dev .
agent-compose -f agent-compose.yml config --quiet
agent-compose -f agent-compose.yml up
./scripts/run-compose-demo.sh
```

上面需要网络/镜像/daemon 访问和明确执行授权。本包未运行这些命令。镜像在 agent-compose daemon 使用的镜像存储中是否可见，须现场检查；不能认为 CLI 机器的 Docker 镜像自动出现在远端 daemon。

合成 demo 在 guest 的 `/tmp/cqa/receipts` 写回执。新沙箱不是共享存储，因此不能将这个 demo 的重放语义说成跨沙箱持久化。正式持久卷、身份和日志通道属于 X1/S2。

不要在 `image:` 写 `${IMAGE}`：所核验的上游文档明确该位置不做变量插值。`agent-compose.yml` 使用字面镜像标签，并由后续验收固定实际 digest。

## 切换真实数据/模型

`configs/live.example.json` 不是可直接运行配置。先审批来源、传输、模型预算；复制到忽略的 `.local/` 并按该文件路径校正相对数据目录，填入可验证的 Search 服务和完整模型端点，显式批准网络。外部凭据存于约定的环境引用，不入 Git。不要把示例域名当上游产品的内置地址。

API 响应中 200 + `NEEDS_REVIEW`/`INSUFFICIENT_EVIDENCE` 是完整的业务结果。外部失败会留 blocked receipt，不能删除记录自动重跑。

## 证据与源码绑定

`evidence/bootstrap/source-manifest.json` 绑定交付包文件，`verification.json` 记录实际环境、检查与未验证项。该环境尚无用户 Git commit，所以不编造 SHA。用户初次提交后，OMP 重新执行检查并把证据绑定真实 HEAD/差异。

## S1-02 Token 用量候选

用户在授权实施 #2 后修订方向：后端使用 OMP 的百智云 provider、`grok-4.6`，当前仅记录 Token 消耗。此修订替代旧 Qwen／10 元／20 次／发送前完整输入费用证明，不是把旧 AC 宣称通过；历史候选及其 BLOCKED 证据保留在 `.local/s1-02/`。用户现已授权提交／推送 S1-02、回写并关闭 #2，实际发布版本和完成回执以 GitHub 为准，不包含其他工单实施或真实调用。

候选复用 Go query CLI、HTTP 客户端与回执，仅参考 OMP 已配置的 `baizhi-chat/grok-4.6` 路由；**不启动 OMP 编码代理、不读取其凭据库、不引入 SDK 或多模型注册框架**。固定生产端点为 `https://ai-api-gateway.app.baizhi.cloud/api/openai/chat/completions`，受控模拟允许 literal-loopback HTTP。默认网络仍关闭，X1 仍只能使用合成资料与原文提取。

### 请求与核验边界

HTTP JSON 仅含 model、messages、stream、max_tokens、response_format 五项。采用固定 `grok-4.6`、非流式、`max_tokens:2048` 和 JSON Object；不发送 Qwen 的 enable_thinking、max_completion_tokens，也不发送该 OMP 路由未启用的 reasoning_effort、store 或工具。2,048 是请求参数，不是已证明的完整计费输出上限；截断／工具调用／结构或引用不合格仍拒绝成功。

messages 仅有固定 system 提示词和 user 数据包；user 仅 question、asOfDate、untrustedEvidence，证据仅 id/title/locator/text。不传 requestId、Case/Invocation/digest、地域行业、来源 URL、业务回执或配置。凭据仅由本服务的 `CQA_LLM_TOKEN` 等配置环境引用注入，**不会复制或自动解析 OMP 的 apiKey**。无重定向、环境代理或自动重试。

OMP 配置声明不等于百智云网关认证。模型真实可用性、JSON 参数兼容、返回模型标识、实际处理地域和条款仍待真实运行前核验；不能把百炼北京或 xAI 直连接口的承诺转给代理。依据见 [百智云／Grok 核验记录](research/s1-baizhi-grok.md)。

### 用量记录与失败

每个新结果的 `tokenUsage` 使用以下状态：

| status | 含义 |
|---|---|
| `not_called` | 本次未调用生成模型，例如本地无依据、待复核或原文提取 |
| `reported` | provider 返回非负整数的 prompt、completion、total 三个计数 |
| `partial` | 只返回部分可识别计数；保留已知项，缺失项不补零或推导 |
| `unknown` | 用量缺失、格式无效，或请求／响应结果无法确认；没有可宣称的完整用量 |

`inputTokens` 原样对应 `prompt_tokens`，`outputTokens` 对应 `completion_tokens`，`totalTokens` 对应 `total_tokens`；可选 `cachedInputTokens`、`reasoningTokens` 保留对应 details 字段。这些不是 OMP 已归一化并扣除缓存的 `input/output/cacheRead`。总量只采用 provider 的 total，不用输入、输出、缓存、推理重新相加，也不强制 total 等于 input 加 output。xAI 示例的推理计数口径存在差异，不能假定 completion 总包含推理。显式整数 0 与缺失／null 不同；负数、非整数、溢出或结构错误的 usage 整体记 unknown，不阻断本来合格的草稿。

成功结果与用量一起保存至 `storeDir/<requestId>.json` 的 `result.tokenUsage`；拒绝输出但已收到可解析用量时，blocked 回执的顶层 `tokenUsage` 仍保留这些计数。HTTP 错误、超时或无法解析响应记 unknown，不等于未计费。只读取回答 content，不保存 reasoning_content；不保留完整 provider 原始响应／错误或密钥。provider/model 标识为固定请求目标，不是第三方实际路由证明。

发送前仍持久化既有 requestId 预留，失败则零发送。保留的 reserved 回执记 unknown；响应后写盘失败或进程崩溃可能丢失已返回用量，不能将本地记录视为完整账单。成功重放返回同一结果及用量、零新增调用；失败／未知同 ID 返回 `PREVIOUS_EXECUTION_UNRESOLVED`。历史缺少 tokenUsage 的回执不补计数。汇总只能按唯一保留回执统计一次，未知项另列；删除、回滚或更换回执目录会丢失记录及去重保护，本版没有另一套额度账本或防删恢复机制。

### 可操作验证与证据

```bash
# 不需要真实密钥；构建真实 CLI，使用临时合成配置和本机受控 HTTP 服务。
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 -v ./internal/agent -run '^TestS1CLI$'
make verify
```

场景覆盖草稿／用量、字节一致重放、冲突、恶意输入、已知用量下的输出拒绝、错误／超时的未知用量、缺失计数不阻断后续请求及网络／存储／X1 边界。细分计数检查覆盖缺失、部分、显式零、无效计数和 provider 总量口径；全部是合成协议验证，不是 Grok 实测 Token。

本轮命令、环境、候选摘要与验收映射固定在私有 `.local/s1-02-token-usage/`，不覆盖历史 `.local/s1-02/`。真实模型调用保持 NOT_RUN；经审核真实来源、账号／地域／条款及最终样例运行授权仍是下一入口。仅记录用量不会限制花费，不可把打开 networkApproved 当作获得运行权限。

### 独立发布候选

S1-02 按实施前快照从混合工作树分离，不夹带尚未提交的 X1 原生 adapter／部署文件。发布树与含 X1 的本地工作树分别验证；原生 LLM 情形按“明确拒绝且零发送”验收，不依赖某一未发布实现的错误码。独立树不支持原生模式，本地 X1 的合成／extractive 限制不变。发布阶段的树摘要、命令和结果另存 `.local/s1-02-publication/`；原有混合候选证据不能替代独立提交树验证。

## S1-01 单文档测试

用户已将本轮范围缩减为保留一份本地测试文档，后续再增加 RAG；不继续来源追查或扩库。

当前文档：`testdata/s1-test-document.json`，同一份测试材料包含第15／16条候选转录。正文注明“仅供软件测试、非现行法规依据”；`synthetic:true`，日期／状态／审核字段全部是合成测试值，不是把代理转录审核成真实法规。配套 `configs/s1-test.json` 固定 local／extractive、关闭网络；`examples/s1-test.json` 使用已确认的全国问题及2026-09-11查询日。回执写入忽略的 `.local/s1-test-receipts/`，检出仓库即可离线运行，不依赖原私有材料。

```bash
make build
./bin/compliance-agent query --config configs/s1-test.json --input examples/s1-test.json
```

已验证输出 `REFERENCE_ONLY`、`synthetic_demo`、一条引用及 `tokenUsage.status=not_called`。原本地验证保存在 `.local/s1-01-test/verification.json`；收口阶段的独立提交树、命令、结果与摘要另存 `.local/s1-01-publication/`。只验证测试资料的本地链路，不证明法规现行性或真实模型能力。

原来源调查和 `.local/s1-01/` 冻结证据保留为历史，不覆盖、不把原真实资料 AC 改记 PASS。默认来源过滤及未知核验日期回归继续保留。S1-01 的提交／推送和 #1 范围回写／关闭不包含 X1 改动或其他工单实施；该离线切片不授予真实模型调用权限。RAG 未实施。

## S1-03 单文档模型验证

`configs/s1-model-test.json` 固定本地测试文档、百智云 `grok-4.6` 和凭据环境引用 `CQA_LLM_TOKEN`，默认 `networkApproved:false`，不自动读取 OMP 配置或创建可联网活动配置。`storeDir` 是模板的回执路径占位，禁网状态下不会写入该目录。模板采用已运行候选的 120 秒本地等待上限；它不是费用控制，也不证明此前未知结果的根因已修复。

```bash
make build
# 离线无依据检查：预期 INSUFFICIENT_EVIDENCE、not_called，退出0。
./bin/compliance-agent query --config configs/s1-test.json --input examples/s1-model-insufficient.json
# 联网闸门检查：预期 NETWORK_NOT_APPROVED，退出1；不读取模型密钥、不调用模型。
./bin/compliance-agent query --config configs/s1-model-test.json --input examples/s1-test.json
```

正常／重放问题来自 `examples/s1-test.json`，无依据输入为 `examples/s1-model-insufficient.json`，均固定2026-09-11、CN／all。每轮授权、候选冻结、一次请求边界和停止规则以 [S1-03 规格](specs/s1.md#s1-03-当前范围单文档真实模型验证)为准，不沿用已结束的调用或凭据读取授权；凭据仅通过 `CQA_LLM_TOKEN` 注入。

获准运行使用独立活动配置和回执位置。遇到失败／未知先停止并保留记录，不通过改 ID 或删除旧回执绕过阻断。

### 固定的运行证据

- [首次单次请求回执](https://github.com/Notyet1307/compliance-query-agent/issues/3#issuecomment-5635408377)：30.034 秒后 `UPSTREAM_OUTCOME_UNKNOWN`；保存 blocked／unknown 回执，未重试或重放，计费仍未知。
- [另获授权后的运行回执与实际草稿](https://github.com/Notyet1307/compliance-query-agent/issues/3#issuecomment-5642521323)：120 秒配置下，28.621 秒得到 `DRAFT_READY`；无依据为 `not_called`，成功重放输出逐字节一致且回执不变。输出保留 `synthetic_demo`、`humanReviewRequired:true`、`entailmentVerified:false`。本次耗时低于原30秒上限，不能把成功归因为延长超时。
- Provider 报告 input 1197、output 183、total 2629、cached input 512、reasoning 1249。保留 provider 原值，不重算 total、不把缓存／推理或重放再累加；这些数字不补齐前次未知用量，也不提供账单或金额保证。
- 本地 `.local/s1-03-run-20260912-01/` 保留活动配置、输入、CLI 输出、回执、`answer.json`、运行摘要及独立 `acceptance.json`。运行代码为 `e3e2bdc3f955b602053cfcea79736f9f42e265e5` 对应的已验证独立树，Go 1.27.1 darwin/arm64；不使用混合 X1 工作树编译物。私有文件不是 GitHub 可下载附件。

人工注意：中央在京／统一定级及分支备案条件完整保留在引用原文，但没有在三条生成说明中展开。用户的接受决定与正式任务状态见 [#3](https://github.com/Notyet1307/compliance-query-agent/issues/3)；决定单独绑定草稿，不改写原结果标记或先前“待审核”的运行摘要。只有这份合成测试文档经过本次验证，不代表真实法规、RAG、上传、OctoBus、guest 或 Accord 验收通过。检出或发布这些文件不授予新的真实调用权限。
