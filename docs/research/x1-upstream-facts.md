# X1 上游只读核验：版本、运行输出与部署边界

日期：2026-09-11。用户批准公开只读核验，X1 优先本机 Docker。本文记录已读取的第一方资料，不是安装、运行或兼容性测试结果。关联候选：[SEED-X1](../issue-seeds/X1.md)；细化规格：[X1 草稿](../specs/x1.md)。

## 1. 固定版本，不混用 main 与 release

| 项目 | 本轮公开查询结果 | X1 建议 |
|---|---|---|
| agent-compose latest 正式发行 | `v2609.2.0`，2026-09-08；annotated tag 解引用至 `043b763f05304c3a55f1bd3f4eb8504a591aa4ba` | 优先该固定稳定版，而非 latest 镜像或浮动 main。 |
| agent-compose 当前 main | `e5cc4b6fa5e1a6ea44c2f381d041001a563bac3e` | 仅作差异调查；其中 workspace mount 是新功能，不能假定稳定版支持。 |
| OctoBus 当前 main | `22bd3865542c23864b30373ec35a9239f91b61ca` | 下文 OctoBus README 事实绑定此提交；不是已部署版本。 |
| OctoBus GitHub latest 正式发行 | `v0.1.0`，2026-06-15，无 release assets | GitHub release、当前源码、npm 与镜像版本必须分别核对，不能直接选 latest 并称已固定。 |

来源：[agent-compose release](https://github.com/chaitin/agent-compose/releases/tag/v2609.2.0)、[tag 解引用](https://api.github.com/repos/chaitin/agent-compose/git/tags/3864f794d3dc55f139f46d7609b51401a2388d5b)、[agent-compose main 提交](https://github.com/chaitin/agent-compose/commit/e5cc4b6fa5e1a6ea44c2f381d041001a563bac3e)、[OctoBus main 提交](https://github.com/chaitin/OctoBus/commit/22bd3865542c23864b30373ec35a9239f91b61ca)、[OctoBus release](https://github.com/chaitin/OctoBus/releases/tag/v0.1.0)。

### 官方 ARM64 镜像元数据

只查询 Docker Hub 的公开 tag API，没有拉取任何层、登录 registry 或运行镜像。

| 镜像 | OCI index digest | linux/arm64 manifest digest | API 所报 ARM64 size |
|---|---|---|---|
| `chaitin/agent-compose:v2609.2.0` | `sha256:a97253d934ac46909a98020c1a900adf2dbe756460511a32fd6f70a7561c685a` | `sha256:d5941e20ad8d465f64bc8e113b3fcaaef7883ed525620c63895bb7014f32e15f` | 250317650 字节 |
| `chaitin/agent-compose-guest:v2609.2.0` | `sha256:f035c09a810098bf38d8d6892ec3e965fae5effd45b6606d171a4cc80d8a3c0c` | `sha256:0cdd66ab19799fb5ea161130bd839cd39713db092f89f1d1fbf4f7689c28e3e8` | 962259676 字节 |

来源：[daemon tag API](https://hub.docker.com/v2/repositories/chaitin/agent-compose/tags/v2609.2.0/)、[guest tag API](https://hub.docker.com/v2/repositories/chaitin/agent-compose-guest/tags/v2609.2.0/)。这是 registry 元数据，不是本机内容校验、签名验证或下载磁盘总量保证。[发行工作流](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/.github/workflows/images.yml) 定义 amd64/arm64 发布；同文件里的完整 Docker 生命周期 image-smoke 运行在 amd64，不替代本机 ARM64 验收。

## 2. command-mode：有结构化外壳，但不是业务 JSON 通道的别名

固定版 [CLI 手册](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/docs/pages/command-line-manual.md) 的 run/exec/logs/inspect 段及实现说明：

- `run <agent> --command <常量文本>` 走 guest `agent-compose-runtime exec`，命令 stdout/stderr 会流式输出并持久化到 Run。`exec` 使用同一命令输出路径，但不创建 ProjectRun；不能用它代替要求 run 引用的业务验收。
- `--json` 抑制流式文字并输出平台结果外壳；`-d --json` 返回初始 run/sandbox 信息，之后用明确 run ID 的 inspect/logs 查询，不按“最新一个 run”找结果。
- command REPL 的每行是一个新调用，不是运行中进程 stdin 透传。固定输入文件仍是保守选择。本试验不使用交互/PTY 分支。
- `--sandbox` 只能复用同 project/agent 的 sandbox。默认完成会停止 runtime；`--keep-running` 明确保活。cleanup 失败可能让 run 停留 running 并带 cleanup_error，不能因命令已结束就认定平台终态成功。

[cli_run_output.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/cmd/agent-compose/cli_run_output.go) 的 `composeRunOutput` 包含 `id, project_id, agent_name, status, sandbox_id, exit_code, output, result_json, logs_path, artifacts_dir, cleanup_error` 等字段。注意 run ID 在 CLI JSON 中叫 **id**。

[execution_transition.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/runs/execution_transition.go) 的 `transitionFromCommandResult` 将 `result_json` 填成命令元数据：

```json
{"mode":"command","command":"受信固定命令","success":true,"exitCode":0}
```

它不是 `cqa.result/v1`。业务结果仍需从完整 `output` 或已核验 artifact 提取；不得从混合日志截取“像 JSON 的一段”。成功命令的应用层约束应为 stdout 只有一个完整 CQA JSON、stderr 无应用诊断；最终由试验验证外壳、命令状态、绑定字段和业务数据。

[cli_run_command.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/cmd/agent-compose/cli_run_command.go) 的 `manualRunClientRequestID` 包含当前 RFC3339Nano 时间；重复执行相同 CLI 命令不具有稳定的客户端幂等 ID。CQA requestId、本地回执与平台 run ID 必须分别记录，不能把它们混称“一次请求”。

[cli_run_stream.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/cmd/agent-compose/cli_run_stream.go) 在本地 context 取消且已知 run ID 时尝试 StopRun，但忽略该停止请求的返回错误；因此 CLI 被取消不证明 sandbox/Search 已停止。X1 使用独立 run ID 观测终态，不自动重发业务查询。

## 3. 日志确实落盘，且权限不是天然 0600

[command_execution.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/runs/command_execution.go) 创建 `command-request.json`，同步到 guest 后执行，再取回 runtime artifacts。[execution_log.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/runs/execution_log.go) 定义宿主 artifact 目录为 sandbox 目录下的 `state/runs/<run-id>`；命令执行使用 `transcript.txt`。目录创建 mode 为 0755，日志文件创建 mode 为 0644，实际权限还受 umask/父目录约束。

结论：CQA 回执文件 0600 不会自动保护平台运行日志。试验需私有数据根目录、只记录合成资料，并实测调用者可见性。run/logs 返回的目录是 daemon 命名空间中的位置，不是假定 Mac 上同路径存在。

CLI 手册的 retention 段还说明：默认相关 TTL 为 0；启用归档后，archive 可含 home/state/context/日志且独立于 sandbox 目录，当前无完整 archive 删除/恢复 API。X1 不启用自动归档/GC，也不使用全局 prune；结束前明确保留/清理本次资源的范围。

## 4. 本机 Docker 的真实权限与路径问题

固定版 [README](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/README.md) 与 [部署手册](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/deploy/README.md)：installer 针对 Linux amd64/arm64；macOS native CLI 是源码构建产物，不是该 release 提供的 Mac 二进制。Linux daemon/standard guest 镜像支持 ARM64；Docker 模式可用于 macOS Docker Desktop，无需因此引入 KVM、BoxLite 或 Microsandbox。

[官方 docker-compose.yml](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/docker-compose.yml) 把 Docker socket 挂给 **daemon**，服务端口映射为 loopback；UI 是独立可选 profile。X1 可优先采用容器化控制平面以避免本机源码构建依赖，但 Docker socket 是控制 Docker Engine 的高权限，不是“只读查询权限”，必须单独审批。此处的 Docker Compose 只启动基础设施，业务查询仍由 agent-compose 的真实 run/guest 执行。

[docker_runtime.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/driver/docker_runtime.go) 的 `dockerSandboxHostConfig` 只显式设置 mounts/ports/network/init/AutoRemove；不能从这段代码推导完整生产沙箱或资源配额。其网络选择会从 daemon 所在网络中排序选一个非 bridge/host/none 网络，失败可能回退 default。需要实际检查 guest 加入了哪个网络，不能假定和 OctoBus 管理面天然隔离。

固定版已有 `agents.<name>.volumes` 的 bind/read_only 字段；workspace 则是 file/git 的私有副本语义。[固定版 YAML 手册](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/docs/pages/agent-compose-yaml-manual.md) 与 [main 的 workspace mount 段](https://github.com/chaitin/agent-compose/blob/e5cc4b6fa5e1a6ea44c2f381d041001a563bac3e/docs/pages/agent-compose-yaml-manual.md) 不同，后者新增 mode/read_only 及共享源拓扑检查。X1 不为省复制而追 main，不把宿主完整仓库作为工作区；只用专用合成输入目录，并验证 daemon 到 Engine/guest 的路径映射。

## 5. 已证实的 OctoBus 与原生配置边界

固定版 YAML 的 octobus_servers/capset_ids 段：

- qualified capset（例如 `enterprise/compliance-readonly`）是完整的 sandbox 授权项，不等于 unqualified `compliance-readonly`。qualified server 名只负责 agent-compose 路由，上游收到实际 capset 名。
- 上游 token 由 daemon 保留，不注入 guest；guide/网关配置缺失或取回失败可能只告警，sandbox 仍会创建。因此启动成功不能证明能力可用。
- sandbox 创建时捕获 capset 授权集；后续调用从当前 agent 配置解析 server URL/token。X1 固定试验期间配置，不能以热重配置增加已有 sandbox 权限。

[OctoBus 固定提交 README](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/README.md) 进一步说明：

- 默认 admin/data/reflection 共用 `127.0.0.1:9000`；容器内默认 `0.0.0.0:9000`。远程访问控制由部署承担。
- capset **默认没有 token**；未加 token 时 public data/OpenAPI/reflection 等接口可访问。添加 token 后才要求 Bearer；capset token 的保护不等于 admin API 保护。
- daemon/service import/start 需要 Node/npm/protoc/git；仅装 OctoBus 二进制不够。官方容器包含常规 runtime 依赖，但拉镜像/导入服务/安装 service dependencies 都是独立副作用。
- `capset add-instance` 默认暴露该实例当前全部方法；预置时应用精确 method selection，不给试验 guest 这些 admin 操作权限。
- Connect 使用 protobuf JSON，拒绝未知字段且默认省略零值。真实合成 Search 回包必须经过实际 proto/SDK 序列化，不能把 httptest 原样 JSON 视为真实契约认证。

本仓库 `protocol/compliance.proto` 没有对应可运行服务。需预置一个真正遵循该契约的合成 Search package；calculator 只能证明基础通路，不能代替 X1 的查询验收。

## 6. 原生 ABI：只有 gRPC，不是另一条 Connect URL

固定版 [OctoBus 集成规格](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/docs/design/octobus_integration.md) 与 [capproxy 实现](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/capproxy/proxy.go) 一致：guest 数据面是 `grpc.NewServer` + raw protobuf passthrough，不提供 HTTP JSON invoke 或 Connect endpoint。

| 项目 | 精确契约 |
|---|---|
| daemon 启动 | 同时设置 `CAP_GRPC_LISTEN` 和 guest 可达的 `CAP_GRPC_TARGET`；变更后重启 daemon、新建 sandbox。控制面 connected 不能证明这两个值已配置。 |
| guest 注入 | `CAP_GRPC_TARGET`、每 sandbox 的 `CAP_TOKEN`；token 用于绑定 sandbox 和其 capset 集，不是 OctoBus bearer。 |
| 固定业务方法 | `/compliance.v1.KnowledgeService/Search`，请求/响应为 protobuf wire。 |
| metadata | `x-capability-sandbox-token: <CAP_TOKEN>`；`x-octobus-capset: enterprise/compliance-readonly`；`x-octobus-instance: compliance-kb-synth`。后两个是候选操作员配置，不由问题或来源决定。 |
| 授权与转发 | proxy 验证完整 qualified declaration 属于 sandbox，解析 target 后转发真实 capset `compliance-readonly`；删除 guest sandbox credential/authorization，注入 daemon 的上游 bearer。 |
| 错误边界 | 未声明 capset 为 `PermissionDenied`；缺 sandbox credential 为 `Unauthenticated`；业务调用缺 instance 为 `FailedPrecondition`。不是靠目标不存在或上游 token 错误实现授权拒绝。 |
| guide | daemon GET `/admin/v1/catalog/{capset}?format=md&grpc=true`，写 `<sandboxDir>/runtime/mpi/catalog.md`，guest 为 `/data/runtime/mpi/catalog.md`；qualified capset 在 guide 中重写为完整 declaration。失败只告警。 |
| discovery | 支持经代理访问 gRPC reflection；固定 method/proto 的 Go 客户端不需要动态反射或解析 guide Markdown。 |

proxy 的 guest 入口在该实现中未启用 TLS，需隔离专用网络；到上游的 TLS 由 target URL/transport 处理。capproxy 只在 sandbox 层限制 capset；具体 instance/method 必须由 OctoBus 的 capset method selection 限制。

因此本仓库现有 Connect HTTP client 不直接兼容原生路径。若选择永久进程内 Go client，需要固定 gRPC/protobuf runtime 与生成代码；不应手写 wire codec。但 X1 可以先复用已有工具，避免提前增加 Go 依赖：

- [固定 standard guest Dockerfile](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/guest-images/Dockerfile.agent-compose-guest) 的构建定义将 grpcurl `v1.9.3` 安装到 `/usr/local/bin/grpcurl`，X1 可先使用这一已有工具；实际镜像内行为仍待验。
- [grpcurl v1.9.3 源码](https://github.com/fullstorydev/grpcurl/blob/v1.9.3/cmd/grpcurl/grpcurl.go) 明确提供 `-expand-headers`：把 argv 中固定字面量 `x-capability-sandbox-token: ${CAP_TOKEN}` 在进程内部展开，官方说明其目的就是避免秘密出现在命令参数中。不得先在 shell/父进程展开。
- 必须显式 `-plaintext` 连接该固定版 capproxy；grpcurl 默认使用 TLS，漏传时的握手失败是传输配置错误，不能算 sandbox/capset 授权拒绝，`-insecure` 也不等于禁用 TLS。`-d @` 从 stdin 读取请求；`-proto`/`-use-reflection=false` 可使用固定仓库协议、无需动态发现；`-max-time`/`-max-msg-sz` 支持时限及接收上限。错误 exit status 为 `64 + gRPC code`，CLI 本身错误另行映射；不解析或公开原始 status 详情。
- 由 CQA 知识 client 内的固定无 shell 子进程调用保持“先 reserve 回执，再 Search，再筛选/输出”的原有顺序；不能先在外面取数据再走 local 模式冒充同一路径。输入/输出仍严格有界；不开 verbose，不把 metadata 打印进日志。

这只节省 X1 的协议依赖/生成成本，不改变线上真实 gRPC 调用，也不证明子进程方案适合永久运行。来源文本/guide 仍不具有指令或权限配置权。以上工具行为为源码核验，镜像内实际执行 NOT_RUN。

凭据持久化亦有独立风险：daemon-wide token 在 [configstore](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/storage/configstore/capability_gateway_store.go) 的 SQLite token 列存储；qualified server token 在受管配置中解析。[API redaction](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/agentcompose/api/project_octobus.go) 不等于静态加密。“留 daemon”是边界陈述，不能升格为 secret-at-rest 证明；需保护专用数据根并避免公开原始数据库/配置。

## 7. OctoBus 运行产物与真正的合成 Search

版本域独立：固定源码树 daemon npm package 为 0.2.0，SDK 为 0.6.0；公开 npm 对应版本的 gitHead 分别为 `2034fb32e45f364f1d9577374b031fd6fc993195`、`d3ec46f7c33d59d8469b4533fead8afe7e9d386a`，都不等于本次调查的 main commit。来源：[daemon npm](https://registry.npmjs.org/@chaitin-ai%2foctobus/0.2.0)、[SDK npm](https://registry.npmjs.org/@chaitin-ai%2foctobus-sdk/0.6.0)、[源码 daemon package](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/npm/octobus/package.json)、[源码 SDK package](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/sdk/package.json)。

README 的 GHCR 示例与实际 [发布 workflow](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/.github/workflows/docker.yml) 不同；workflow 默认发布 `docker.io/chaitin/octobus` 的 latest/ref tag，平台 amd64/arm64。本轮 [Docker Hub main tag API](https://hub.docker.com/v2/repositories/chaitin/octobus/tags/main/) 返回：

- OCI index：`sha256:a8160fc126e1c3814ea7287a9b527cf88e2a9b08f08ac839757fbd67ad223aec`。
- linux/arm64 manifest：`sha256:58ead4734ad2772ca21299b47fea71f057e4cea2a9b9a72c6234bc75dcad4264`，API size 167189855 字节。
- [INFERENCE] 该 index 对应 `22bd386…`：push 时间与该提交 [成功的 Docker workflow](https://github.com/chaitin/OctoBus/actions/runs/34108448521) 相符，但未拉取或验证 provenance。它只能作为待确认候选。授权执行后必须核对实际 `octobus version` 的 version/commit（workflow 预期为 `main` / `22bd386`），不能把推断升格为本机观察。

[Dockerfile](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/docker/Dockerfile) 的最终镜像包含 daemon 和 Node/npm/protoc/git 等工具，但没有 SDK 或任何 services。合成 Search 还需真实 package；导入会准备 production dependencies，不等于“镜像里已经离线可用”。选择 exact SDK `0.6.0` + lockfile，授权 npm 获取或由操作员预先审核 bundling artifact；`--offline` 不会补齐不存在的缓存。实际镜像内工具版本与导入行为均 NOT_RUN。

按 [service-package contract](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/docs/design/technical/service-package.md) 和 [第一方 service 实现](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/services/arxiv__api/src/service.js)，最小候选由以下内容组成：

- 根 `package.json`：ESM、bin 名与 `service.json.name` 一致、唯一直接生产依赖 `@chaitin-ai/octobus-sdk: 0.6.0`。
- `service.json`：`schema=chaitin.octobus.service.v1`、`name=compliance-kb`、`proto.roots=["proto"]`、`proto.files=["proto/compliance.proto"]`。单次 unary 可选 on-demand，无需常驻搜索服务。
- 原仓库 `compliance.proto`、合成 corpus、Node bin；`defineService` 注册 `compliance.v1.KnowledgeService/Search`，由 `runServiceMain` 启动。handler key 不含前导 `/`，gRPC method path 则含 `/`。
- operator 导入并创建 `compliance-kb-synth`，建立 `compliance-readonly`，使用 `--no-all-methods` 后只 select Search。package 执行/依赖脚本属于受信代码预置，guest 不获得 import/instance/capset admin 能力。

## 8. 单 token 的权限取舍：不能默认为“只读管理凭据”

第一方证据：

- agent-compose [capability/client.go](https://github.com/chaitin/agent-compose/blob/043b763f05304c3a55f1bd3f4eb8504a591aa4ba/pkg/capability/client.go) 使用 target token 访问 admin status/list/catalog GET；同一 target 的 token 被 capproxy 注入数据面。
- OctoBus [admin/admin.go](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/internal/admin/admin.go) 对除 status 外的所有 admin 路由统一验证 admin token；没有只读 scope 分支。[store.go](https://github.com/chaitin/OctoBus/blob/22bd3865542c23864b30373ec35a9239f91b61ca/internal/store/store.go) 将 admin token 与 capset token 分表存储；各表为空时对应认证不要求 token。

若 admin 和 capset 同时启用认证，单个 agent-compose target token 必须在两边都有效。将同一 secret 登记为 admin token 与指定 capset token 是可行配置，但 daemon 因此持有**该隔离 OctoBus 实例的管理权限**。上游设计中“control plane read-only”描述它调用的 API，不是 token 被强制限制只能 GET。

另一配置是 admin 无 token、capset 有 token，但这把管理面的保护全部交给网络隔离，不能描述为更小授权或默认安全。本轮不选择无认证管理面。

该取舍随后已由用户以“仅合成试验接受”确认，权威范围见 [ADR-X1-01](../specs/x1.md#adr-x1-01隔离合成试验的-daemon-凭据)。此许可与 Docker/P1 执行许可分开，不授权查询智能体调用 admin 写接口，也不批准生产方案。Direct Connect 或修改上游鉴权都不是自动替代路径。

## 9. 本轮证据边界

本文件以上内容记录公开研究阶段的 GitHub/raw/npm/Docker Hub 只读核验，不以源码推断代替运行证据。随后用户批准并完成 P0 与一次 P1–P3/S1–S6 合成试验，实际结果见 [X1 执行记录](../specs/x1.md)；本节不再表示当前仍全部 NOT_RUN。

### 后续实测修正

- 固定 OctoBus 镜像内 `version=main, commit=22bd386`；Node `v22.22.1`、npm `9.2.0`、protoc `3.21.12`，SDK exact `0.6.0` 与 38 个 bundled production dependencies 已离线导入并真正调用。npm lifecycle 脚本未执行。
- 初次预检 guest 的 `grpcurl -version` 实际为 `dev build <no version set>`，不能把 Dockerfile 的 v1.9.3 当作二进制自报版本；`security-gate.json` 测得 SHA256 为 `db2411793f99e78e6d084797c51dded8caeee49405fe3e75070e6df14ae2a4f7`。此测量绑定初次派生镜像 c85a87a2…；最终 6ba9c7aa… 复用同一固定 base，但未另测其 grpcurl SHA，不把前者写成后者的独立测量。绝对 proto 路径必须同时提供 `-import-path /opt/cqa/protocol`。
- 无模型的 command 使用 `provider: codex`；`pi` 在启动阶段仍要求有效模型。输入挂载使用 daemon bind 的子目录 `/x1/inputs/requests`，而非 bind 根本身；后者在本次路径映射中失败。
- 实测 stop/resume 可更换容器并丢失 `/tmp` 回执；显式删除 runtime 后的恢复被上游 UUID guard 拒绝。原生调用可用不代表回执跨重建持久化。具体命令、失败保留与限制在 X1 记录中。
