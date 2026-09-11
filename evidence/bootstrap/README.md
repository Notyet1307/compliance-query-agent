# 启动包验证证据

本目录记录 ChatGPT 生成环境的真实执行，不等于用户 Mac、agent-compose 沙箱或客户环境已通过验收。

## 当前结果

`make verify` 最终执行成功：28 个 Go 顶层测试；含 9 个子用例时共 37 个测试通过事件，测试级 0 失败、0 跳过；CLI 包没有 Go 测试文件，由实际进程检查覆盖。另实际运行 20 项 CLI/本机 HTTP 进程检查。Go 格式检查、`go vet`、构建和 `go test -race` 通过。

实际编译器为 **Go 1.23.2 linux/amd64**；目标 Go 1.27.1 的验证为 **NOT_RUN**，不得把旧编译器成功作为生产工具链建议。测试中的 HTTP 上游是 loopback httptest，不是真实 LLM/OctoBus。

## 证据入口

- `verification.json`：环境、命令、结果、限制及源码绑定。
- `source-manifest.json`：交付源码、配置、数据、文档的 SHA256 清单；不编造 Git commit。
- `go-tests.jsonl`：Go 原始测试事件。
- `smoke.json`：真实 CLI/HTTP 进程的逐项结果。
- `race.txt`、`race-exit-code.txt`：实际竞态检测结果。
- `vet.txt`、`build.txt`、`gofmt.txt`：成功时无输出的原始日志。

命令执行过程中曾修正媒体类型校验的编译错误、统一 smoke 的实际错误码，并在冷编译超出执行窗口后重跑；这些不是未解决问题。这里只把最终源码上实际完成的执行记为 PASS，不把中断尝试计算为通过。

真实模型、OctoBus、agent-compose、Accord 和客户环境未联调。完整限制见 `verification.json`。原生 OctoBus guest 适配和 Accord 角色接入是未实现，不只是缺一个凭据。

## 复核

在根目录运行 `make verify`；新证据写入忽略的 `.local/verification/`，不会覆盖本次历史证据。提交到用户仓库后，OMP 重新执行并绑定实际 HEAD/工作树差异。源码清单只提供内容绑定，不是可信签名或独立审计。
