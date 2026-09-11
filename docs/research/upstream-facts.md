# 上游核验记录与限制

核验日期：2026-09-10。来自当前用户提供的三份手册/提示词，以及以下上游材料的只读检查。**仅阅读，不代表执行兼容性认证。**

## 用户提供的工作流材料

- `操作手册(1).md`：分工、真实框架、Spec/GitHub 权威、候选切片、Matt 适配、接管/独立验收和记忆边界。
- `07_向ChatGPT申请设计启动包.prompt.md`：初始设计交付项；未决事项不得写成批准决定；未执行标 NOT_RUN/BLOCKED；不替用户选择可见性/许可/外发策略。
- `01_新项目接管.prompt.md`：首轮接管输出要求；保留工作树；不自动开发/发布。

## Accord

仓库： https://github.com/Notyet1307/Accord

读取基线：`2668ee62f249d462930bd980c8178ce5f24f7f6e`。

具体读取：README；Issue/PR 最近更新列表；src/profile-runtime.ts（前 170 行）；docs/specs/r003-live-driver.md（前 160 行）。

- https://github.com/Notyet1307/Accord/blob/2668ee62f249d462930bd980c8178ce5f24f7f6e/docs/specs/r003-live-driver.md
- https://github.com/Notyet1307/Accord/blob/2668ee62f249d462930bd980c8178ce5f24f7f6e/src/profile-runtime.ts

事实：治理/执行分离；R003 四固定 Profile 和冻结合成 Case；正式来源接受/角色输出和发布都受现有契约限制。接入合规 Agent 需要批准的新接缝，不是仅修改 prompt。

没有克隆/构建整个 Accord，没有运行其 CI 或 live qualification，没有修改文件/工单。

## agent-compose

仓库： https://github.com/chaitin/agent-compose

观察到 main ref：`e6857fdfa11b0fab30de8116ffae11aa4ff42559`。下列 README/manual 通过当时 main 读取；该 ref 是调查时观察值，不据此伪称下载了完整固定版本。YAML manual blob：`3ddf0a37b14c7e84b9ea97fefe944a01ca06a99b`。

- https://github.com/chaitin/agent-compose/blob/main/docs/pages/agent-compose-yaml-manual.md
- https://github.com/chaitin/agent-compose/blob/main/docs/pages/command-line-manual.md

事实：daemon + CLI；guest/provider；run --command；严格 YAML；image 不插值；原生 octobus_servers / capset_ids；qualified capset 不等于 unqualified；原生 token 留在 daemon；guide 失败可能仅告警继续创建沙箱；command 输出会进入 run 记录。

影响：固定命令 + 数据通道；不能把任意 Go 服务镜像当合格 guest；不得以 sandbox 创建成功证明数据可用；真实数据输出也需要日志治理。agent-compose 的上游 README 显示 AGPL-3.0 许可标识，实际采用版本的完整许可/交付影响由用户专项审核；本次没有为新仓库指定许可证。

## OctoBus

仓库： https://github.com/chaitin/OctoBus

README 当时 main 的 blob：`23f143b96013428a3f5bbbe68f0b8c03aab7be68`。

- https://github.com/chaitin/OctoBus/blob/main/README.md

事实：service/instance/capset/method binding；unary Connect/gRPC/MCP；Connect 示例形状 `/capsets/dev/connect/calculator-test/calculator.v1.CalculatorService/Add`。公开端口/远程 TLS/网络访问控制由实际部署承担。

本项目只借用协议形状；KnowledgeService/Search 是新建领域 contract，不是上游已实现服务。

## LLM 与 Go

- https://api-docs.deepseek.com/api/create-chat-completion/
- https://go.dev/dl/

只采用 chat-completions 非流式、messages、model、response_format、finish_reason 的基本协议；未指定厂商或模型、未实际调用。所选模型能否接受 JSON 输出/参数仍须真实验证，不静默切到另一协议或模型。

Go 官方下载页显示目标 1.27.1；生成环境只有 1.23.2。本包分开记录兼容下限、推荐目标和实际执行环境。

## 调查限制

容器侧直接访问 GitHub 的一次尝试因 DNS 不可用失败；仓库事实来自可用 GitHub 连接器/网页读取，不依赖失败的下载。未取得本机 OMP、真实三平台环境/凭据，因此对应验证明确 BLOCKED，而不是被 mock 替代。

本次没有完成真实法规版本与文本使用权核验，因此语料全部 synthetic。知识治理文档是后续建设计划，不冒充现行法规清单。
