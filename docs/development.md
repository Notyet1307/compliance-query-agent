# 开发、运行和验证

## 工具链和依赖

建议目标 Go `1.27.1`，来自 2026-09-10 查阅的 Go 官方下载页；`.go-version` 用于声明目标，不会自动安装。`go.mod go 1.23.0` 为语言兼容下限。生成环境现有 Go 为 `1.23.2 linux/amd64`，只对该环境实际测试。目标版本与用户机器运行均 NOT_RUN。

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
