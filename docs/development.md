# 开发、运行和验证

## 工具链和依赖

目标 Go `1.27.1` 由 `.go-version` 声明，不会自动安装；`go.mod go 1.23.0` 是语言兼容下限。Go `1.23.2 linux/amd64` 仅是启动包的历史生成环境；本机 Go `1.27.1 darwin/arm64` 已实际验证，X1 guest 为 `linux/arm64`，证据见 [接管回执](handoff-receipt.md) 与 [X1](specs/x1.md)。

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

## agent-compose 本地合成演示

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

上面是通用环境准备命令，不是任意机器的运行保证；需要网络/镜像/daemon 访问和明确执行授权。X1 已使用固定 digest、私有 Docker 配置和 ARM64 guest 完成真实 command/原生 Search，实际命令与差异见 [X1 执行记录](specs/x1.md)。CLI 机器上的镜像不等于远端 daemon 可见。

guest 在 `/tmp/cqa/receipts` 写回执。X1 发现同 sandbox stop/resume 也可能更换容器并丢失该目录；新 sandbox 不共享回执。正式持久卷、身份和恢复策略仍需批准；不要删除 blocked 回执或复制旧回执伪造重放。

不要在 `image:` 写 `${IMAGE}`：所核验的上游文档明确该位置不做变量插值。`agent-compose.yml` 使用字面镜像标签，并由后续验收固定实际 digest。

## 切换真实数据/模型

`configs/live.example.json` 不是可直接运行配置。先审批来源、传输、模型预算；复制到忽略的 `.local/` 并按该文件路径校正相对数据目录，填入可验证的 Search 服务和完整模型端点，显式批准网络。外部凭据存于约定的环境引用，不入 Git。不要把示例域名当上游产品的内置地址。

API 响应中 200 + `NEEDS_REVIEW`/`INSUFFICIENT_EVIDENCE` 是完整的业务结果。外部失败会留 blocked receipt，不能删除记录自动重跑。

## 证据与源码绑定

`evidence/bootstrap/source-manifest.json` 与 `verification.json` 仅绑定历史启动包。后续本机证据必须绑定各自 Git 基线与候选快照；X1 的授权、候选 manifest、实际命令、场景、日志扫描和停止状态保存在私有 `.local/x1-runtime/evidence/`，索引见 [X1](specs/x1.md)。这些证据不是 Accord trusted qualification。

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

## S2-01 单文档检索离线候选

以下保留 S2-01 初次离线候选记录，不代表当前状态。后续真实仅查询权限和有界目录兼容证据、原 R2 资源保留失败、独立 retain 修正及其限定接受均已分别留存；当前查询通路又在 S2-02 最终组合候选复核，见下方当前验收节。历史失败不改成 PASS，S2-01 整票接受不从后续调用成功推导。

```bash
# 在未准备过的检出目录中生成隔离材料；已有输出时拒绝覆盖，不删除旧记录重跑。
python3 experiments/x1/prepare.py --trial s2-01
python3 experiments/x1/prepare_test.py
make verify
# 默认禁网闸门：预期 NETWORK_NOT_APPROVED、退出1，不读取模型凭据。
./bin/compliance-agent query --config configs/s2-native-test.json --input examples/s1-test.json
```

准备入口复用 X1 服务源与固定协议，默认不带参数仍生成原 X1 数据；`--trial s2-01` 生成私有 `.local/s2-01/prepared/`：单文档服务源码、固定正常／无依据请求、禁网配置、0700 回执及独立观察目录。请求保留 S1 问题／范围／日期，但使用 S2-01 身份。包内 `data/demo-corpus.json` 保留既有文件名，内容是同一 S1 测试文档，不是原五域演示 corpus；服务包名称仍标识复用的 X1 测试组件，不冒充生产知识服务。

`configs/s2-native-test.json` 保持 `networkApproved:false`、extractive 与原生 X1 严格测试边界；它不是已经可联网的活动配置。后续 guest 读取固定 `/s2/inputs/config.json` 和请求子目录，回执指向 `/s2/receipts`，须先验证真实宿主／daemon／guest 映射。准备程序不安装依赖、不调用 daemon、不创建模型权限。

服务的 `CQA_OBSERVATION_ROOT` 仅由受信操作者设置，不取自 RPC 输入；未设置时保留 X1 原路径。S2 运行须使用独立目录并核对映射，不能让 S2 调用混入 X1 或本地 SDK 的事件记录。计数仅记录进入／完成、请求摘要及时间，不把原始问题写入该观察日志。

### 已执行的软件验证

- `make verify`：目标 Go 1.27.1 darwin/arm64，37 个测试根、80 个含子测试通过事件；既有 CLI／loopback smoke 通过，新增准备检查纳入基础入口。
- 准备检查证明使用正确单文档、保留原问题并换独立身份、默认禁网、回执私有、重复准备不覆盖已有 blocked 回执及原 X1 数据选择不变。
- 复用原 X1 已冻结 tgz 中的 SDK 0.6.0／锁定 bundled dependencies，在项目私有暂存目录展开，未下载或执行 npm 安装／生命周期脚本。实际 Node 26.5.0／protoc 35.1 的 SDK CLI 成功返回唯一测试来源及正确正文摘要，非法 limit 被拒绝；独立观察目录只留下正常调用的 entered／completed。
- 真实 CQA CLI **以 local 模式消费 SDK 返回的 corpus**：正常 `REFERENCE_ONLY`、无依据 `INSUFFICIENT_EVIDENCE`、未知核验／重叠版本 `NEEDS_REVIEW`、篡改 `SOURCE_DIGEST_MISMATCH`、完成重放逐字节一致且回执不变；模型均未调用。原生模板禁网闸门另验。这里不是把“先取资料再走 local”冒充完整原生链路。
- 已交叉编译 linux/arm64 guest 二进制并固定 guest 构建上下文；未构建或运行容器镜像。固定服务 tgz 可供后续离线导入，实际 OctoBus 导入与权限认证仍 NOT_RUN。

本轮证据在 `.local/s2-01/`：`baseline.json`、`sdk-reuse.json`、`sdk-check.json`、`offline-cli-check.json`、`artifacts.json` 和最终 `verification.json`。所有结果绑定混合工作树候选，不把 S1 发布 SHA 单独冒充本次构建身份。X1 旧包、旧回执与冻结证据保留。

### 待批准的真实运行包

`.local/s2-01/runtime-authorization-request.json` 固定三个历史镜像 digest、服务包／guest 二进制及输入摘要。`runtime-proposal/` 中的联网活动配置是**待授权材料**；它不改变源码模板的禁网默认，也不是运行许可。镜像当前可用性、派生镜像构建与平台 YAML parser 未验证，缺镜像／依赖即停止，不自动下载。

拟授权范围：仅本机已核实的 OrbStack；从固定上下文离线构建 guest；新建专属 daemon、OctoBus、一个受管 guest 及两个 internal 网络；仅受信 daemon 持 Docker socket（具有本机 Docker 控制权限，不能称作仅查询权限）。生成独立测试凭据，OctoBus 管理凭据只用于操作者预置，daemon 仅持查询凭据及其独立控制面凭据；不读取 OMP 密钥、X1 token 或旧数据库。服务离线导入若需要依赖准备，只能使用已捆绑的锁定依赖，禁用生命周期脚本，不下载。

运行包包含正常、完成重放、无依据、未授予 capset、无效能力凭据、受控取消后 blocked 重放、末尾服务不可用七类固定场景，最多八次计划内 query CLI 执行，**零模型调用**。这是一份人工执行清单，不是软件次数闸门。查询 token／目录流程不兼容、网络／挂载越界或计划外失败／未知立即停止，不升权或自动重试；仅按确认归属停止本次新建资源，不删除数据／证据，不操作 X1 或其他项目。每项真实结果另记，离线通过不代替 G2。

## S2-02 受管模型与持久回执离线实施

本节记录当时获准的离线实施，不是新的执行许可。复用 S2-01 的固定单文档、协议和服务包；其有界 G2、原 R2 保留失败及已接受的 retain 修正分别保留在私有 `.local/s2-01/delivery-1/`。后续 S2 真实运行与用户接受见下方当前验收节；不复用旧回执或操作权限。

`configs/s2-managed-test.json` 新增显式 `octobus_native_s2`／`managed` 测试边界，默认 `networkApproved:false`。只允许 synthetic + LLM + 固定 `grok-4.6`；旧 X1 仍拒绝 LLM，正式资料模式不因本变更接受 synthetic。此文件不是可直接联网的部署配置。

### 运行绑定与存储

- `managed.daemonOrigin` 固定受信 daemon origin，`project`／`role` 绑定同机信任范围；它们不是本服务签发的权限，也不是已证明存在的平台环境变量。`OPENAI_BASE_URL` 必须逐字匹配该 origin 下 `/api/runtime/sandboxes/<64位十六进制ID>/llm/openai/v1`，仅追加 `/chat/completions`；不接受查询参数、凭据 URL、其他协议路由或任意来源选出的地址。明文 origin 仅支持专用 `cqa-s2-02-daemon` 或 literal private／loopback IP，实际网络仍需单独批准。
- 仅从 `OPENAI_API_KEY` 读取平台代理凭据；CQA 不读取百智云上游密钥。CAP gateway 的主机须与批准 daemon 一致，继续使用固定 Search、proto、无 reflection 的子进程。真实平台须选择 Chat Completions 兼容配置；旧 Codex／Responses 槽位不算兼容。Token 的 provider／wire 权限由真实 daemon 验证，本地 URL 校验不证明其权限或上游路由。
- configDigest 仍绑定全部稳定配置、凭据引用和资料配置，另绑定实际运行文件 SHA256，而非只用 Version 字符串。解析后的 sandbox URL、CAP 连接端口及同权限 token 值不进入摘要。构建或稳定配置变化不能重放；平台运行引用由执行证据单独记录，不回写已完成业务结果。
- `/s2/receipts` 必须由操作者预置为同项目／角色专用的 0700 宿主持久 bind，guest 使用固定绝对路径。启动读取 Linux `/proc/self/mountinfo`，要求精确、可写、非 tmpfs／ramfs／overlay 的挂载及无 symlink 的私有目录；不自动创建替代目录。预留与完成前还检查原目录身份和权限。Mac 本机不提供此内核表，受管模式明确报 `PERSISTENT_STORE_REQUIRED`，不退回普通目录。
- 挂载表只证明当前 namespace 的挂载形状，不证明宿主 backing 正确或数据跨重建存活；这仍属于真实 G3。目录身份检查也不是对恶意管理员的竞态隔离或删库恢复。
- 新完成回执记录 `resultDigest`，受管回放必须匹配；损坏／缺摘要／不合法状态拒绝，reserved／blocked 保持未解决。历史非受管完成回执缺摘要仍按原身份条件读取，不回写补字段。结果摘要只防意外损坏，不是签名或防管理员篡改。响应后写盘失败保留原 reserved／unknown，可能丢失已返回用量，不自动重试。

### 可操作离线检查

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 -timeout=90s ./internal/agent -run '^TestS2'
make verify
# 默认禁网：预期退出1、NETWORK_NOT_APPROVED；不读取凭据。
./bin/compliance-agent query --config configs/s2-managed-test.json --input examples/s1-test.json
```

S2 检查使用原生子进程 double、内核挂载表 fixture 和 loopback 模型服务；分别计数模拟 Search／代理接收／上游转发。覆盖草稿与完成重放、地址／凭据轮换、真实不同测试可执行文件导致冲突、稳定配置／关联冲突、来源闸门、挂载及目录替换、回执损坏、预留／完成写失败、拒绝／超时／取消后不重试与并发单一发送。既有 S1 CLI／客户端检查继续覆盖最小外发、恶意正文、输出拒绝、Token 四态和隐私边界。它们不是 agent-compose／OctoBus／Grok 实测，也不证明平台无内部重试。

私有离线候选、命令、环境和摘要保存在 `.local/s2-02/offline-1/`；实际 Darwin CLI 可演示禁网／缺挂载拒绝及同文档本地查询与重放。软件检查不是 G1–G3 真实证明；后续实际运行与用户接受单独列于下节，不回写原离线记录。

## S2 当前真实查询／草稿验收

用户已接受本次具体测试草稿（“满足预期”）。公开 [验收摘要](../evidence/s2-query/accepted-summary.json)、[原样业务草稿](../evidence/s2-query/draft.json) 和 [本次发布验证](../evidence/s2-query/verification.json) 分别记录 live／人工结论、实际输出与离线检查，不公开 `.local/` 原始日志、数据库或凭据。

成功尝试执行四个固定场景：未授予 capset、无效能力凭据均拒绝且未进入 Search；无依据返回 `INSUFFICIENT_EVIDENCE` 且不生成；正常返回 `DRAFT_READY`。实际 Search 入口 2 次、模型请求 1 次，五个认证后审计事件均绑定实际原生 run／sandbox。只验证固定 synthetic 单文档；原样保留 `humanReviewRequired:true`、`entailmentVerified:false` 和 provider 用量，不推导法规效力、账单或模型内部行为。

受管运行依赖固定 `platform-fix-3` 定制 agent-compose，而非未经修补的上游镜像二进制；它修正 command 凭据 run 归属及空 `OPENCODE_MODEL` 覆盖正式 model 字段的问题。上游来源、实际源码／二进制摘要和补丁范围在验收摘要中明确列出。原始 daemon／guest 镜像均使用只读二进制覆盖，不能用镜像 digest 代替实际执行文件身份。本次不向上游仓库提交、不分发平台二进制，也不改变本仓库许可证。

三层最小上游补丁按 [平台候选清单](../integrations/agent-compose/candidate.json) 顺序应用到固定 `043b763f05304c3a55f1bd3f4eb8504a591aa4ba`，仅补丁保留其 [上游 AGPL 许可](../integrations/agent-compose/LICENSE.txt)，不移植到 CQA 包或替本仓库选许可证。本次离线应用核对中，1397 个候选文件有 1395 个逐字节一致；另两份 `pb.go` 是上游归档缺失的生成输入，需遵循上游生成流程，预期摘要已列出。补丁应用成功不是重新构建或平台全套验收。

公开可直接演示的入口仍是 `make verify`、`make demo` 和本地 CLI；受管真实草稿不是一键重跑包。`prepare.py --trial s2-01` 可生成共用单文档服务源码／输入，但不安装 SDK、不打包服务或导入 OctoBus。实际运行还需单独准备固定 SDK 服务包、受控输入挂载、查询认证和 Chat Completions 配置，并使用包含固定路径 `grpcurl` 的批准 guest 基础镜像。现有 Codex 原生提案不是 S2 活动配置；私有服务归档、镜像、上游生成工具／依赖均未随本次源码发布分发或安装。

平台最小回归先失败后通过；七包 race 命令整体失败，六包通过，调度包 `TestSchedulerLLMAsyncRejectsFanOutBeyondOutstandingLimit` 在未改父候选与修正版均失败。当前试验关闭 scheduler，未据此宣称平台全套通过。原 run 归属失败、启动失败和操作者遗漏 `control.json` 导致的 unknown 均保留；后者修正为运行前预置并核对既有 `delayMs:0` 控制文件，不增加探针或重试。

本次没有执行完成结果再提交、跨容器／新 sandbox 重放或 live 并发。G3／A07／A08／S2-02-AC5 延期，现有去重、私有挂载和 unknown 不重试保护保留；当前业务切片完成不等于整票／整 S2 完成。新建容器已停止保留，后续运行须新授权。
