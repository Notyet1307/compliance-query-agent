# 接管核验回执

日期：2026-09-11。范围：接管核验与设计补缺；不实施业务功能。结论：**启动包的离线合成查询链路已在本机目标 Go 上通过；真实知识质量、外部平台接缝与生产资格未通过本轮验收，也未获联调授权。**

## 1. 产品复述与本轮权限

一个独立的安全合规查询智能体，内部拥有业务逻辑和 LLM 接口。知识范围保留信息安全等级保护、关键信息基础设施保护、安全政策、行业知识、商用密码应用安全性评估；五域共用一次有界查询流程，不是五个互相对话的 Agent。

目标分工不变：agent-compose 承载运行；OctoBus 提供受限外部查询通路；未来由 Accord 以受管角色调用，保留其 Case、冻结上下文、来源接受、审批和发布权。本项目只输出资料参考、候选草稿和未解决项，不判企业“合规通过”、不认定关基、不代替等保/密评结论，也不建设聊天平台或 Controller。

本轮先解释检查副作用，再取得用户选择：

- **批准临时副本验证**：复制非秘密启动包文件，隔离缓存，运行原样 `make verify` 和正常/无依据/冲突三例；允许短暂 loopback 监听、临时进程、合成回执和本轮证据。
- **仅本地材料**：不联网重新查询上游。真实 LLM、OctoBus、Docker/agent-compose、Accord 联调均未授权；未安装依赖、下载工具链或镜像、探查 daemon、读取客户数据或凭据。
- 唯一新增的正式文档是本回执；其余本轮文件在独立 `.local/handoff-lxooa_y9/` 内。未修改业务代码、测试、Spec 验收预期、AGENTS、CLAUDE、既有配置或 seeds；未发布工单、初始化 Git、commit/push/merge/release。

## 2. 目录、指令与候选基线

### 2.1 Git 事实

启动命令：

```text
pwd && git remote -v && git branch --show-current && git rev-parse HEAD && git status --short --untracked-files=all
```

实际输出：

```text
/Users/yet/Developer/compliance-query-agent
fatal: not a git repository (or any of the parent directories): .git
exitCode=128
```

因此此处是已保存的启动包目录，不是可识别的 Git 工作树。remote 查询失败；后续分支/HEAD/status 因 `&&` 未执行，不能编造分支、提交 SHA、远端或“工作树干净”。没有扫描父级/其他仓库寻找替代仓库，也没有自行建立 `.git`。

接管前已经存在 `bin/compliance-agent`、`.local/receipts/`、`.local/verification/`，未将其结果当作本轮证据，也未覆盖它们。对这几处仅核验目录元数据：`.local/receipts` 为 `0700`，其余三个目录为 `0755`；没有读取既有回执正文。忽略规则覆盖 `.local/`、`bin/`、`.env*`（保留 `.env.example`），但忽略规则不是数据访问控制，且当前尚无 Git 仓库。

### 2.2 生效指令与阅读顺序

本会话注入的仓库 `AGENTS.md` 是已加载约束；本轮用户授权进一步限制写入和联网范围。按要求依次读取 handoff → product → MVP Spec → CONTEXT → architecture → development → docs/agents 三份适配 → seeds → 实际代码与测试，并补读安全、知识治理、已有 ADR 和工作入口。

`docs/agents/issue-tracker.md`、`domain.md`、`skill-usage.md` 已复用；没有选择 triage、没有新增标签体系，没有把 Skill 默认发布/提交动作当授权。`ADR-0001` 仍为 PROPOSED，MVP Spec 仍为 DRAFT_FOR_OWNER_REVIEW，所有 seeds 仍是未发布候选；本回执不擅自改变这些状态。

### 2.3 无 Git 时的摘要绑定

以 `evidence/bootstrap/source-manifest.json` 中列出的 68 个文件为明确范围，读取当前字节并生成本轮独立 manifest。当前 68 个文件与原 manifest **全部一致**；这证明包内一致性，不证明上游来源真实性或安全资格。

| 证据 | 值 |
|---|---|
| 本轮目录 | `.local/handoff-lxooa_y9/` |
| 实际验证工作目录 | `.local/handoff-lxooa_y9/snapshot/` |
| 本轮源码 manifest | `.local/handoff-lxooa_y9/source-manifest.json` |
| 本轮 manifest SHA256 | `09397040f353ebe02bc25ad8cf59c333ae2af66a21a79e41e91d9830b01b8097` |
| 本轮构建二进制 SHA256 | `401fa4207e61656eb7a6ea4fd9d40e0e01866319845267788abd3f3b31fc28f1` |
| 源码 Git SHA | 不可用：当前目录无 Git；未创建提交 |

副本复制上述 68 个文件，另复制 bootstrap manifest/verification 供历史证据核对；不复制既有 `bin/`、`.local/` 或真实环境文件。构建完成后验证副本中这 68 个源文件仍与本轮 manifest 相同。新回执不包含在测试前的源码 manifest 中；不要改写旧 manifest 来伪造启动包原始状态。

## 3. 真实模块、合成样本、mock 与提案

| 分类 | 事实、入口与限制 |
|---|---|
| 已实现并本机运行 | `cmd/compliance-agent/main.go:23-94`：`query` 单次 JSON CLI、`serve` loopback HTTP；不是 UI。CLI 可从固定文件或进程 stdin 读取，成功 stdout 输出 JSON，错误 stderr 输出固定错误码、退出 1。 |
| 已实现业务流程 | `internal/agent/engine.go:61-134`：请求校验 → 持久预留 → 检索 → 资料校验/筛选 → 提取或生成 → 引用集合校验 → 保存结果；完成重放不重新查询或生成。 |
| 已实现检索资格 | `knowledge.go:23-158`：本地 corpus 严格解码/原文 SHA256、topic/地域/行业过滤、关键词子串排名、有效区间与核验日期、同文档同定位重叠版本阻断、最多选五片段。不是语义搜索或跨法规冲突裁决。 |
| 已实现回执/API | `store.go:20-120`：私有目录、O_EXCL 预留、原子完成、未知不自动重试；`http.go:14-68`：loopback、Bearer、输入大小、错误码与重放响应头。不是跨系统去重或租户权限。 |
| 合成数据 | `testdata/demo-corpus.json`：10 个虚构片段，五域正常例及过期核验、重叠版本、未来/历史例；dataset 为 `synthetic-starter-20260910`。所有来源为 `synthetic:true`、`fixture://`，没有真实法规标准。 |
| 真实提取而非 mock LLM | `clients.go:99-107` 的 extractive generator 逐字返回所选原文。默认 `configs/demo.json` 为 local/extractive、`networkApproved:false`。 |
| 客户端已实现，兼容性仅 mock | `clients.go:24-91,116-193`：直接 Connect unary Search 和非流式 chat-completions。`agent_test.go:321-446` 用本机 httptest、合成 token/响应验证形状、错误、截断、伪造引用与不重试；不证明厂商或实际 OctoBus 兼容。 |
| 纯领域协议提案 | `protocol/compliance.proto` 是拟建 KnowledgeService/Search，不是 OctoBus 内置知识系统；没有凭这个文件获得可用服务实例。 |
| 部署材料/未接入 | `agent-compose.yml`、guest Dockerfile、原生 capset YAML：材料存在，不代表实际 parser 接受、镜像可运行或 Go 已消费注入通道。 |
| Accord 提案 | `integrations/accord/role-proposal.json` 不是已存在的注册 API 或已获准 Profile；没有 Accepted Evidence、审批或发布能力。 |

当前输入为 `cqa.query/v1`；显式给 topic、日期、CN 地域、industry。`types.go:143-219` 严格处理 JSON 与请求；`caseId/invocationId/contextDigest` 只验证形状并绑定回传。HTTP 200 可以表示完整的“无依据/待复核”业务结果，不能据此认定资料正确。

## 4. 输入到输出的三例与异常边界

三例都在本轮新编译的实际 CLI 上运行，退出码均为 0，stderr 均为空；不是从测试预期或旧输出复制。共同结果：`schemaVersion=cqa.result/v1`、`dataMode=synthetic_demo`、`generationMode=extractive`、`humanReviewRequired=true`、`entailmentVerified=false`。

### 4.1 正常：有可引用的虚构片段

输入 `examples/mlps.json`：问题“演示材料中的资产清单需要记录什么？”，`requestId=demo-mlps-001`、topic=mlps、asOfDate=2026-09-10、CN、telecom。

路径：命中“资产清单”关键词；`demo-mlps-assets` 的行业 all 可用，公布/生效日期为 2026-01-01、核验日期覆盖查询日；通过筛选后原文提取。

实际输出摘要：

```json
{
  "status": "REFERENCE_ONLY",
  "claims": [{
    "text": "【虚构测试材料，不是法律或标准】演示资产清单包含资产名称、责任人和用途。该约定仅用于本软件测试。",
    "evidenceIds": ["demo-mlps-assets"]
  }],
  "reasonCodes": []
}
```

citation 的原文与 claim 相同，SHA256 为 `e03df83209102a6319cc46a0f15b9be9c7599045390f75589b73a77c5e19f846`。完整输出：`.local/handoff-lxooa_y9/mlps.stdout.json`。对应 `knowledge.go:90-158`、`clients.go:99-107`、`engine.go:107-134`；运行检查为 `TestFiveDomainsAndExceptions/mlps` 和 smoke 的 `cli_mlps`。

### 4.2 无依据：不补出审计日志保留数字

输入 `examples/insufficient.json`：问题“截至查询日期，三级系统审计日志应保留多久？”，其余范围同上、独立 requestId。

路径：没有匹配可用片段，`engine.go:103-105` 在 generator 之前返回；本机 `TestNoEvidenceDoesNotCallLLM` 还通过替代 generator 的计数验证未调用生成器。

```json
{"status":"INSUFFICIENT_EVIDENCE","claims":[],"citations":[],"reasonCodes":["NO_ELIGIBLE_EVIDENCE"]}
```

完整输出：`.local/handoff-lxooa_y9/insufficient.stdout.json`。这仅表示当前资料不足，不表示现实中没有相应要求。

### 4.3 冲突：不让模型挑一个版本

输入 `examples/conflict.json`：问题“请查询重叠版本测试”，topic=mlps、查询日仍为 2026-09-10。

路径：`demo-conflict-v1/v2` 同时命中，documentId 均为 `demo-conflict`，locator 均为“演示冲突条目”，有效区间重叠且 version/hash 不同；`knowledge.go:124-139` 阻断，`engine.go:99-101` 在生成前返回。

```json
{"status":"NEEDS_REVIEW","claims":[],"citations":[],"reasonCodes":["OVERLAPPING_SOURCE_VERSIONS"]}
```

完整输出：`.local/handoff-lxooa_y9/conflict.stdout.json`。这是同文档同定位、已命中候选内的版本检查，不是任意文档之间的语义冲突识别。

补充：现有 smoke 实际运行 `examples/stale.json` 得到 `NEEDS_REVIEW`；Go 的 stale 子测试同时断言 `CURRENCY_UNVERIFIED`。未来条目和历史结束日边界也已通过现有测试。示例日期固定在 2026-09-10，未改为本轮日期；更换查询日会改变核验资格，不能悄悄更新样本日期以维持通过。

### 4.4 可操作演示

在当前保留的副本中执行（会重放本轮已完成的合成回执，不会联网）：

```sh
(
  cd .local/handoff-lxooa_y9/snapshot
  ./bin/compliance-agent query --config configs/demo.json --input examples/mlps.json
  ./bin/compliance-agent query --config configs/demo.json --input examples/insufficient.json
  ./bin/compliance-agent query --config configs/demo.json --input examples/conflict.json
)
```

不要删除回执来制造重试。改输入/配置但沿用 requestId 会冲突；新 requestId 也不自动产生联网或费用权限。

## 5. 本机环境、命令与验收证据

### 5.1 工具链和副作用

| 核验 | 实际结果 |
|---|---|
| `uname -srm` | `Darwin 25.5.0 arm64` |
| `.go-version` | 目标 `1.27.1` |
| `go version`，`GOTOOLCHAIN=local` | `go version go1.27.1 darwin/arm64`，与目标一致 |
| `go.mod` | `go 1.23.0` 是语言兼容下限；仅标准库，无第三方模块 |
| `python3 --version` | `Python 3.14.6`，未安装第三方 Python 包 |
| `make --version` | GNU Make 3.81 |
| 工具路径 | Go/Python 为 `/opt/homebrew/bin/go`、`/opt/homebrew/bin/python3`；make 为 `/usr/bin/make` |
| `file` 本轮二进制 | `Mach-O 64-bit executable arm64`；不是 Linux guest 二进制 |
| `/usr/local/bin/docker --version` | `Docker version 29.4.0, build 9d7ad9f`；只验证 CLI，不连接 daemon |
| `which agent-compose octobus` 对应 PATH 查找 | 初始组合查找没有这两个路径；仅说明当前 PATH 未发现 CLI，不证明全机未安装或服务不存在 |

环境证据：`.local/handoff-lxooa_y9/environment.json` 与 `verification.json`。隔离 PATH 不含 `/usr/local/bin`，故保存 Docker 版本时改用已知绝对路径；没有扩大环境扫描。Python 的 platform 描述另记为 `macOS-26.5.2-arm64-arm-64bit-Mach-O`，不拿它替代上表的实际内核/架构命令。

原样 `make verify` 的副作用经批准后仅发生于副本：写 bin、`.local/verification`、测试临时文件/回执，执行 Go test/vet/build、Python 静态检查及真实 CLI/loopback smoke。没有修改源码的 formatter 调用，`gofmt -l` 只检查。

执行使用干净的显式环境，不继承个人 provider token：

```text
PATH=/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOENV=off GOTELEMETRY=off
HOME=<本轮目录>/home
TMPDIR=<本轮目录>/tmp
GOCACHE=<本轮目录>/gocache
GOMODCACHE=<本轮目录>/gomodcache
PYTHONDONTWRITEBYTECODE=1 LANG=en_US.UTF-8
```

这是受控本地验证设置，不是 OS 级网络隔离证明。`make verify` 的脚本及测试已读取；网络路径仅用本机 loopback。smoke 的服务进程终止并通过清洁退出断言，合成 token 未反射到其 stdout/stderr。

### 5.2 本轮实际执行

开始时间：`2026-09-11T00:36:39.077554+00:00`。cwd 为上面的 snapshot，命令为 `make verify`；**退出码 0，stderr 为空**。

| 检查 | 结果 | 本轮证据（相对 `.local/handoff-lxooa_y9/`） |
|---|---|---|
| `gofmt -l cmd internal` | PASS，无待格式化文件 | `snapshot/.local/verification/gofmt.txt` |
| `go test -json -count=1 ./...` | PASS：28 根测试、37 含子测试通过事件，0 失败、0 跳过用例 | `snapshot/.local/verification/go-tests.jsonl`、`checks-detail.json` |
| CLI package 的 go test 事件 | 无测试文件；不是一个被跳过的测试用例，另由 smoke 覆盖进程 | `checks-detail.json` |
| `go vet ./...` | PASS | `snapshot/.local/verification/vet.txt` |
| `go build -trimpath -o bin/compliance-agent ./cmd/compliance-agent` | PASS | `snapshot/.local/verification/build.txt`，二进制摘要见上 |
| `python3 scripts/repo-check.py` | PASS：15 JSON，4 本地 Markdown 链接，无断链；不检查上游 YAML parser | `snapshot/.local/verification/repo-check.json` |
| `python3 scripts/smoke.py --binary bin/compliance-agent` | PASS：20 项真实 CLI/loopback 检查 | `snapshot/.local/verification/smoke.json` |
| 三例 CLI + 输出断言 | PASS：状态/原因、空证据、原文 SHA256、合成标记与人工复核标记 | 三个 `*.stdout.json` 和 `*.stderr.txt` |
| `python3 scripts/check-bootstrap-manifest.py` | PASS：68 文件，mismatches=[] | `bootstrap-manifest-check.json` |

汇总原始 stdout/stderr：`make-verify.stdout.txt`、`make-verify.stderr.txt`；结构化汇总：`verification.json`。这里的 68 文件 checksum 检查不是 Accord 的合成来源 manifest，也不是 trusted qualification。

`evidence/bootstrap/verification.json` 仍只代表生成环境的 `Go 1.23.2 linux/amd64`；其 race PASS 不迁移为本机 PASS。本机已解决目标 Go/Mac 未运行这一缺口，但未重新检查官方最新版本、安全公告或生产部署工具链。

### 5.3 对应 MVP A01–A12

| AC | 本轮判定 | 已执行证据与不能扩大解释的部分 |
|---|---|---|
| A01 五域 | PASS（合成软件行为） | `TestFiveDomainsAndExceptions`、五个 CLI 正常例 |
| A02 无依据 | PASS | `TestNoEvidenceDoesNotCallLLM`、无依据 CLI；没有真实模型调用 |
| A03 日期/重叠版本 | PASS（既有规则） | 五域异常、`TestHistoricalInterval`、`TestUnreviewedSourceBlocks`；无跨法规语义裁决 |
| A04 原文完整性 | PASS | `TestDigestTampering`；不证明发布者身份/资料合法性 |
| A05 synthetic 禁用 | PASS（`allowSynthetic=false`） | `TestSyntheticDenied`；“live”不是独立不可绕过的运行身份，见设计缺口 |
| A06 引用集合 | PASS | `TestModelCitationValidation`、`TestModelRejectsForgedCitationAndDoesNotRetry`；非语义蕴含证明 |
| A07 重放与冲突 | PASS（本地存储） | Idempotency/ConcurrentReservation/Restart/ConfigDrift 与进程重启 smoke；不保证跨沙箱或所有并发请求同时成功 |
| A08 未完成不重试 | PASS（现有本地场景） | `TestReservedNeverRetries`、`TestDeadlineAndReceipt`；不是外部执行已取消的证明 |
| A09 网络配置边界 | PASS（配置及 loopback mock） | NetworkApprovalAndTLS/NoRedirect/LoopbackHTTP；没有生产 egress 隔离证明 |
| A10 本地 API | PASS | ServerAuthAndLoopback/APIRejectsOversizedBody 与真实 HTTP smoke |
| A11 两类 HTTP 客户端 | PASS（mock contract only） | ModelHTTPContract/OctobusConnectContract/Truncation；真实兼容性 NOT_RUN |
| A12 CLI/API 进程 | PASS（本机合成） | smoke 20 项，包括重启重放、401/415/413、重复 key、关闭与 token 反射检查 |

本机 race、官方漏洞核验、Linux guest、agent-compose parser/daemon、真实 OctoBus/LLM、Accord E2E 均 **NOT_RUN**。后五类真实联调另有 **BLOCKED：缺授权、已核验运行环境/接口与凭据边界**；不能被上表 PASS 覆盖。

## 6. 代码与设计之间必须保留的缺口

1. **知识资格不等于知识真实。** SHA256 只验证所给原文，`curatorReviewed/validityCheckedAt` 是管理员记录；没有许可证明、核验人签名、来源维护流程或语义正确性证明。MVP 正常例不足以回答真实等保问题；S1 仍需授权资料与人工答案。
2. **检索与冲突范围有限。** 关键词子串匹配先决定候选；同文档同定位的重叠检查只覆盖命中的候选。漏召回的版本或跨文档义务冲突不会被识别。不能以“有引用”代替覆盖率评估。
3. **`live` 不是代码中的独立安全状态。** `configs/live.example.json` 推荐关闭 synthetic；实际拒绝由 `allowSynthetic=false` 控制（`config.go:75-115`、`knowledge.go:34-40`）。mock 正是通过允许 synthetic 来测试网络客户端。后续受限真实部署必须由受信配置/验收固定这一开关，不能把文件名或 networkApproved 布尔值当权限。
4. **原生 capset 与 Direct Connect 不是同一路径。** Go 目前只支持 local/octobus_connect，直接从环境取上游 token 后加 Bearer（`engine.go:26-38`、`clients.go:24-37`）；没有原生 discover/注入代理适配。不能把原生 YAML 的“凭据留 daemon”属性套给现有 Direct 客户端。
5. **command-mode 的本地 CLI 接口不等于跨沙箱传输。** 本机 `--input -` 可读 stdin，但资料明确提醒 command REPL 非 stdin 透传；实际受信输入 artifact、输出提取和 run 终态仍须按固定运行版本验证。不得把问题或检索正文拼成 shell 命令。
6. **日志是独立数据出口。** CLI stdout 含 claim、引用原文与关联字段；本地材料记载 command 输出进入 run 记录。因此即使 OctoBus token 留 daemon，答案和片段仍可能持久化到运行日志；可见性/保留/删除未核验。
7. **本地回执不是跨沙箱/Accord 身份。** configDigest 包含解析后的绝对路径、配置、Agent 版本和本地 corpus 摘要；不同挂载路径也可能产生配置漂移。同一 store 可重放，空的新 store 不能阻止再次执行。关联字段是输入自报并回传，不证明真实 Invocation、冻结上下文或 caller 权限；目前没有独立 attemptId 字段。不能承诺 exactly-once。
8. **失败未知不等于停止。** 本地 SIGTERM/context 取消只说明本地停止等待；外部 Search/LLM 是否执行或计费仍可能未知。reserved/blocked 没有自动恢复命令，不能删回执重跑；Accord 的 Invocation/Attempt 权威不能迁入本项目。
9. **不静默推进设计状态。** `ADR-0001` 的 Go 固定流程、首条等保切片、原生通路和角色名称仍是建议；用户确认的产品职责不能被这些实现选择反向缩小。

## 7. OMP 与 Matt Skills 的实际安装和加载

### 7.1 OMP：版本可证，当前会话全部覆盖值不臆测

`which omp` → `/opt/homebrew/bin/omp`；`readlink` → `../Cellar/omp/18.1.17/bin/omp`。精确读取该安装包的 `.brew/omp.rb`、`INSTALL_RECEIPT.json`、`sbom.spdx.json`，未扫描其他软件或个人配置：

- 安装版本 `18.1.17`、arm64；`omp --version` 在隔离 HOME/副本中实际返回 `omp/18.1.17`，退出 0。
- 二进制 SHA256：`1c310974d4be8de4e5b7285e9c52647321b412c219790fe0243cf804d5c9d5cc`，与本机配方/SBOM 的 darwin-arm64 发行产物摘要一致。配方来源为 can1357/oh-my-pi release；本轮未访问该 URL，未核验最新发行版或远程签名。
- Homebrew 安装记录 `runtime_dependencies=[]`；安装的是单独可执行文件，不是此处可读的 OMP 源码树。不能由此断言其内嵌依赖无漏洞。
- 本机另有 Bun `1.3.14`、Node `v26.5.0`，路径均在 `/opt/homebrew/bin/`；未将它们当本项目 Go 构建依赖，也未据此要求重新安装 OMP。

使用当前 harness 的本地 `omp://settings.md`、`config-usage.md`、`skills.md`、`context-files.md` 查阅加载规则，没有发起网页请求。文档说明项目设置从 cwd 的 `.omp/config.yml` 加载，但 CLI overlay、运行时覆盖仍可优先。本轮未读取个人全局配置、auth store、环境秘密或当前会话持久日志。

在保留原配置的副本、隔离 HOME 下实际执行：

```text
omp config get memory.backend --json        → value="off", type="enum", exit=0
omp config get autolearn.enabled --json     → value=false, type="boolean", exit=0
omp config get autolearn.autoContinue --json → value=false, type="boolean", exit=0
```

证据：`.local/handoff-lxooa_y9/omp-version.txt`、`omp-config-help.txt`、`omp-project-settings.json`。这证明本机版本识别这些键，且隔离执行的有效值符合仓库配置；**不冒充当前正在运行的会话没有额外覆盖，也不证明 memory=off 会关闭会话/运行日志**。不需要重写 `.omp/config.yml`，不进行 config set/reset。

### 7.2 Matt Skills：可读、可发现、已执行是三件事

通过已命名的 `skill://...` 精确读取并用 `realpath` 定位，十个技能实际入口均为 `/Users/yet/.agents/skills/<name>/SKILL.md`。仅访问这些具名技能及其直接引用文件，没有遍历 HOME 或其他仓库。

| 技能 | 核验事实/依赖 |
|---|---|
| setup-matt-pocock-skills | 可读；prompt-driven 配置流程，不是安装脚本；五份 tracker/domain/triage 模板均存在。本仓库已有适配，不执行 scaffold。 |
| grill-with-docs | 可读；直接依赖 grilling + domain-modeling，两者可读且在当前自动技能列表中。 |
| to-spec / to-tickets | 可读；依赖已有 issue-tracker/domain 配置。默认发布并贴 ready-for-agent，与本轮授权冲突，故不执行。 |
| implement | 可读；依赖 tdd 和 code-review，默认末尾 commit 被仓库/用户授权禁止。 |
| tdd / code-review / diagnosing-bugs | 当前列表可发现，磁盘文件存在；TDD 的 tests.md/mocking.md 存在。本轮没有声称运行一次完整技能工作流。 |
| domain-modeling / grilling | 可读；domain 的 CONTEXT/ADR 格式依赖存在。术语和 ADR 不因读取技能而自动改写。 |

前五个技能的 frontmatter 都有 `disable-model-invocation:true`；setup 的 `agents/openai.yaml` 另有 `allow_implicit_invocation:false`。它们不在本会话自动技能清单中，但 `skill://` 能读取；OMP 本地技能文档也将该标志归为隐藏/禁止模型隐式调用。**不能因列表缺名就判断未安装，更不能为“修复”而重复安装。**

十个文件均未声明 version 字段，已知 setup 目录也没有独立版本元数据；在不查全局安装账本或上游仓库的范围内，Matt 上游 release/commit **未核验**，不编造统一版本号。用内容摘要固定本机实际版本：

- `.local/handoff-lxooa_y9/skill-manifest.json` 列明十个绝对路径、逐文件 SHA256、显式调用标志和九个直接 Markdown 依赖的存在性；九项均存在。
- 该 manifest SHA256：`2356e3132770041863592c66f1c3f2826179d1094767c812dacc09d4ef36b7c8`。
- setup SKILL.md SHA256：`310fb1a73c0467e617e17d7c41d4a2278b5405c4a27f36d7cd22bb4599aee6bb`。

补缺建议只需保留本回执中的显式调用与版本说明；现有 `docs/agents/skill-usage.md` 已覆盖不发布、不自动提交、不把标签当授权，无需再造一套适配。仓库没有选择 triage，不因安装包中存在 triage 模板就启用它。

## 8. 外部配置与 Accord 基线

本节严格区分“本轮读取到本地文件”与“这些文件记载的历史上游观察”。当前用户选择仅本地材料；没有访问 GitHub、其他仓库、远程凭据或服务。

| 对象 | 本轮可证事实 | 未核验范围 |
|---|---|---|
| agent-compose | `agent-compose.yml:3-12` 使用 provider=pi、Docker driver、字面 `compliance-query-agent-guest:dev`，无挂载/调度/OctoBus 配置；固定脚本拒绝所有参数，只执行常量命令。 | 当前 CLI/parser/daemon 版本、guest ABI、镜像可见性/架构/digest、实际 command 行为。 |
| guest 镜像 | `deploy/Dockerfile.guest:3-8` 要求操作者提供 BASE_GUEST_IMAGE，仅复制二进制、合成配置/资料/一个例子，不改 ENTRYPOINT/CMD。guest config 的回执目录为 `/tmp/cqa/receipts`。 | 基础镜像未固定/构建，未证明 Linux ARM64 或 amd64 可运行；dev 标签不满足正式不可变部署。 |
| 原生 OctoBus | proposal YAML 使用 `octobus_servers.enterprise`、qualified capset `enterprise/compliance-readonly`；变量只读名字，未读取值。 | 注入变量名、代理协议、权限/guide 失败语义和 token 留 daemon 都未实测；不捏造 Go 客户端入口。 |
| Direct/LLM 示例 | live.example 使用 `.invalid` 域名、占位模型，networkApproved=false、allowSynthetic=false、环境引用 token。 | 没有可用端点/模型/实际权限，不能把开关改 true 就运行。 |
| Accord | 本地材料及 role JSON 都引用 `2668ee62f249d462930bd980c8178ce5f24f7f6e`；`isAccordImportFormat=false`、`requiresAccordSideWork=true`。 | 实际当前 main、Issue 开闭状态、批准 Spec、完整代码与门禁执行，均 BLOCKED（本轮无远程读取授权）。 |

`docs/research/upstream-facts.md` 的历史 agent-compose ref 为 `e6857fdfa11b0fab30de8116ffae11aa4ff42559`，同时明确手册当时通过 main 读取；不能当作完整固定版本源码证据。OctoBus 仅记录 README blob `23f143b96013428a3f5bbbe68f0b8c03aab7be68`，不是本机服务版本。历史材料还记录 agent-compose AGPL-3.0 标识；实际采用版本的许可与交付影响仍待用户审核，不由本轮代选许可证。

Accord 的约束按用户确认及 `integrations/accord/README.md:5-38` 保留：

- 四固定 Profile：RESEARCHER、ANALYST、REVIEWER、WRITER；业务“合规资料研究员”不是第五个可直接注册的 Profile。
- 既有冻结合成 Case、合成 source manifest、确定性来源接受、精确人工审批和单一发布者门禁不能绕过。本项目的 bootstrap checksum manifest 与这个来源门禁不是同一物。
- `InvocationBoundOutputContract` 的泛型命名不说明任意领域角色可插拔；历史材料限定的 GenericProfile 仍是 REVIEWER/WRITER。
- cqa 的 source ID 不是 Accord BoardEntry ID；citations 只能先成为候选 EvidenceRef，不能直接写 Accepted Evidence 或把原 ID 填进 sourceRefs。
- 受信适配器将来必须校验 Case/Invocation/contextDigest、Profile/版本/schema、运行引用和旧上下文；无依据/待复核只投影问题或未解决项，运行结束不等于 Case 完成。

未来获准时只读核验的明确入口是上述 Accord 仓库的 main ref、当前 Issue/PR 与实际批准 Spec/相关源文件；先记录真实 SHA 和范围，再讨论 S3。没有读取权限时保持 BLOCKED，不用旧 README 开发指针或本仓库 role JSON 代替批准。

## 9. 建议下一个任务：有界 X1 接缝试验

**推荐只批准一个 SEED-X1，不同时启动 S1/S2/S3。** 本节是补充试验设计，未修改 seed 或 AC，也不是执行授权。

唯一问题：在选定 agent-compose 版本/guest/单一 Docker driver 上，固定命令能否读取受限请求文件，经原生 OctoBus 通路调用一个预置的合成只读 Search，返回可绑定的结构化结果，并明确日志及跨沙箱回执边界？

### 9.1 进入条件与权限包

1. 用户批准特定 CLI/daemon/OctoBus 版本、基础 guest 镜像 digest、目标架构、允许连接的地址/端口和镜像许可；当前本机 ARM64 不推出 daemon/guest 也是 ARM64。
2. 操作者先提供已授权的合成 Search 实例和只读 capset。X1 不授予 Agent OctoBus admin API，也不通过注册服务来绕过“只读”；若实例不存在，明确 BLOCKED，由操作者另行准备。
3. 只允许启动/停止该试验的有限资源，挂载单独合成输入与本次输出位置，不挂 HOME、完整仓库、Docker socket 或客户资料。安装/拉取镜像是独立副作用，批准版本和来源后才可做。
4. 固定本仓库候选 SHA；尚无 Git 时先用本回执的摘要副本作为明确试验输入，不能因此自行初始化或提交。正式跨仓库交付仍须建立批准的 Git/Spec 基线。
5. **不调用真实 LLM**：采用 extractive 输出。上游 token 仅由操作者经约定 secret reference 配置；不得在回执记录值。先确认确切原生注入契约，再写最小临时适配，不猜变量名、不提前改业务源码。

### 9.2 输入、输出、身份与观测契约草稿

- **输入**：受信发起方从固定目录提供单个 `cqa.query/v1` JSON，遵守现有 65536 字节限制；只含合成问题。固定命令只引用固定配置/文件，问题、source 文本、可变路径不进入 shell 文本。注入正文含引号/换行/命令样式也必须保持数据。
- **传输**：先验证所选版本实际支持的 staging/artifact 路径；若只有 command REPL 而没有可验证文件通道，不伪装 stdin 已打通，记录阻塞。已有 `run-compose-demo.sh` 只证明常量脚本存在，不证明动态输入能力。
- **输出**：从明确的 artifact 或已核验的 stdout 通道提取一个完整结果，分开 stderr、命令回显和平台日志；同时检查进程 exit、run 终态、schema、请求/配置/资料摘要以及引用 hash。run=completed、末尾一段看似 JSON 或 healthz 成功都不能代替结果验收。结果丢失/截断/多个文档保持失败或未知，不回退解析“最像答案”的片段。
- **绑定**：试验台记录固定的 requestId/inputDigest、合成 caseId/invocationId/contextDigest 与平台真实 project/agent/sandbox/run 引用。回包必须对应发起方预期；这个检查由受信试验台完成，不把 guest 自报字段当身份认证。后续真实 Invocation/Attempt 仍归 Accord 所有。
- **回执**：先验证同 sandbox 同 store 重放；再有意使用新 sandbox/空 store 测量是否重复调用。默认不引入共享持久卷；若需要共享卷，另批访问控制与固定挂载路径，不以“都叫 requestId”承诺跨沙箱 exactly-once。
- **日志**：只在本试验所属 run/sandbox 的明确位置检查 stdout/stderr、平台 run 记录、权限、保留/删除和结果引用。记录哪些原文会被持久化；不能全盘扫描 daemon 日志或客户文件。用独立合成 canary 测反射，报告命中与否而不打印 token；精确反射测试不冒充通用 DLP。

### 9.3 最小执行矩阵与停止条件

固定一个版本组合、一个 driver、一个 Search、一个正常合成请求，最多两个 sandbox、五次受控场景；不自动重试失败场景、不做多驱动比较：

| 场景 | 应记录的可观察结果 |
|---|---|
| 1. 正常受限查询 | 真实原生调用一次；结构化 REFERENCE_ONLY、synthetic 标记、精确来源 hash、输入与 run 绑定；单纯 sandbox 创建成功不算。 |
| 2. 同 sandbox 同请求重放 | 原样结果，Search 调用计数不增加；同 ID 换输入的离线检查应拒绝错绑，不偷偷当新尝试。 |
| 3. 未授权 capset | 实际数据方法被拒，不能降级到 Direct/另一个 capset；guide 失败只告警仍启动时，业务也必须不返回伪成功。 |
| 4. 重建 sandbox/空 store | 记录旧回执是否存在、Search 是否再次执行、run 引用是否可找回；如再次执行，结论就是“不提供跨沙箱去重”，由外层 Invocation 策略补齐，不改成 PASS 去重。 |
| 5. 查询中取消/输出未取回 | 分开本地取消、平台终态和上游是否已执行；无法证实时保留 UNKNOWN，不自动重发或接受晚到错绑结果。 |

前三项须打通输入、权限与结果；第四、五项允许得出有证据的“不支持/未知”能力边界，不允许将其伪装为已解决。任何未预期外发、越权、凭据进入 guest/日志，或必须使用未经批准的 host/admin 权限，立即停试验并保留脱敏证据。

交付仅为固定版本/源码或快照摘要、命令、五场景结果、请求/平台引用、脱敏日志定位与临时接缝 Spec/ADR 草稿。X1 完成不代表真实 LLM/资料/Accord 接入通过。

若原生路径无法供 Go 消费，先记录实际协议缺口。Direct Connect 是**需重新批准的备选**：实现较直接，但 token 进入 guest/进程环境，需要额外认证、只读 ACL、egress 与日志治理；不允许试验自动切换，也不同时挂两条相同能力通路。

## 10. Seeds 保留/修改建议与少量用户决策

### 10.1 Seeds：只给建议，未更改文件或发布状态

| Seed | 建议 |
|---|---|
| B0 | 保留。本轮可作为接管证据：本机目标 Go、三例与代码解释已完成；Git SHA、Matt 上游版本和 Accord 最新事实明确有限。候选文件不被擅自标成已发布/关闭。 |
| X1 | 保留为唯一下一任务。批准时补入第 9 节的单 driver/单方法/无真实模型、日志取回、取消/重建矩阵和停止条件；预置服务/镜像/权限是进入条件，不偷偷变成开发平台任务。 |
| S1 | 保留五域范围与首条真实等保闭环。资料使用权、维护责任、人工答案和模型外发预算须先明确；20–30 例仅建议，不升级为已批准 AC。资料遴选本身不严格依赖 X1，可在单独批准后先做准备，但本轮不改依赖或启动它。 |
| S2 | 保留。正式 ticket 应明确依赖“X1 接缝结论 + S1 业务资料”，虽当前 X1 经 S1 间接前置；只实现一条获准通路，不重复挂 Direct/原生能力。持久化/取消/错绑检查必须在此切片完成，不能推给 R0。 |
| S3 | 保留跨仓库批准要求。先只读核验 Accord 当前 main/Issue/Spec；在其批准的新接缝下接入，不替换 R003、扩大固定 Profile 或绕过 source manifest。 |
| R0 | 保留为候选固定后的独立验收；不能是前三个切片第一次跑真集成的场所，不能改候选源码或预期来变绿。 |

所有 Seed-ID 继续只是去重标识，不是 Issue 编号；没有 remote 时不能查重发布，不新增 ready 标签或第二份正式任务状态表。

### 10.2 只需用户判断的关键事项

| 决策 | 推荐与代价 |
|---|---|
| 是否授权上述 X1 及其确切环境/资源范围 | 推荐原生路径优先、合成 Search、无真实 LLM、最多两个 sandbox。代价是先由操作者准备可用测试实例/镜像并审批运行、网络及日志；未准备则只做有证据的阻塞结论，不扩成通用 Runtime。 |
| 原生不适配时，是否接受 Direct 的凭据边界变化 | 推荐先停并看 X1 证据，不预先授权 fallback。接受 Direct 可减少适配工作，但失去“上游 token 不进 guest”的目标，需要重新承担认证/ACL/egress/轮换及日志风险。 |
| S1 的真实资料与模型传输边界 | 下一业务阶段推荐一条有授权原文和人工答案的等保闭环，再扩其余领域；用户/资料负责人决定使用权、更新责任、模型端点、预算和保密。当前无需为整个平台一次性选完模型或知识库。 |

Git 仓库归属、remote、可见性、许可证与提交/发布方式仍由用户决定；它们在“要建立 Git/正式发布”时再审批，不为本轮报告擅自创建远端，也不把这些行政选择塞成多个预发布工单。

## 11. 接管验收与下一入口

| B0/本轮要求 | 判定 |
|---|---|
| 产品职责、真实能力/mock/提案区分 | 完成，见第 1、3、6、8 节；未改产品边界。 |
| 正常/无依据/冲突实际输出及代码/检查对应 | PASS，见第 4、5 节及本轮输出文件。 |
| 目标 Go 与本机检查 | PASS（Mac ARM64、Go 1.27.1、合成链路）；不迁用生成环境或 race 证据。 |
| OMP/Matt 实际路径、版本、依赖/加载 | 完成有界核验；OMP 版本可证，Matt 内容版本可证但上游 commit 未核验，当前会话全部覆盖值未读取。 |
| Accord 最新 main/Issue/Spec、真实四平台接入 | BLOCKED/NOT_RUN，缺本轮授权与已核验环境；保留既有门禁。 |
| command-mode/日志/身份/回执设计补缺 | 已给有界 X1 草稿；原生通路能力仍未实测，不算集成 PASS。 |
| 用户文件与验收预期 | 既有业务/测试/配置/Spec/seeds 不变；本轮独立快照和本回执不覆盖旧产物。 |

下一验收入口：本回执 → 本轮 `verification.json`/源码与技能 manifest → `docs/issue-seeds/X1.md` 和第 9 节的拟议试验范围。若用户批准，先核对该任务的版本、资源和秘密引用边界，再开始 X1；不会由本轮本地验证授权推导外部运行权。

**到此停止，等待用户批准下一步。** 不自动改 seeds、发 Issue、实现 Adapter、运行外部系统或提交发布。
