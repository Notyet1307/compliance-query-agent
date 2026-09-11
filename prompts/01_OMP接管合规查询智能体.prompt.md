这是 ChatGPT 生成、由我保存到仓库的安全合规查询智能体启动包。本轮只做接管核验和必要设计补缺，不直接实现所有功能。

产品已确认：范围包括信息安全等级保护、关键信息基础设施保护、安全政策、行业知识、商用密码应用安全性评估。智能体独立包含业务逻辑和 LLM 接口，由 agent-compose 承载，通过 OctoBus 调用外部系统，未来以受管角色方式加入 Accord。不要改成独立聊天平台、通用 Controller 或五个互相对话的 Agent。

先识别当前目录、真实 Git remote/分支/HEAD/工作树与生效指令；保留全部用户改动。按 AGENTS.md → docs/handoff.md → docs/product.md → docs/specs/mvp.md → CONTEXT.md → docs/architecture.md → docs/development.md → docs/agents/ → seeds 与实际代码阅读。

重点核验：
- 明确当前真实能力、合成样本、mock 协议、纯提案各是什么；用正常/无依据/冲突例子解释输入到输出，并对应代码和可运行检查。
- 检查 OMP 与 Matt Skills 的真实版本、路径、依赖和指令加载；复用 docs/agents 配置，只补缺，不重新覆盖 AGENTS/CLAUDE。
- 说明本地检查的副作用并取得必要审批，再执行 make verify；检查目标 Go 与本机实际版本差异，不把生成环境 Go 1.23.2 的证据当本机结果。
- 真实 LLM、OctoBus、Docker/agent-compose、Accord 联调均未授权。只检查可见的版本/配置事实，涉及凭据、网络、启动服务、安装依赖时先说明边界，不扫描其他目录或秘密。
- 当前引用的 Accord 基线为 2668ee62f249d462930bd980c8178ce5f24f7f6e；有获准读取途径时重新核验实际 main/Issue/Spec。四个固定 Profile、合成 source manifest 和既有门禁不能被本仓库角色提案绕过。
- 原生 capset 模板不等于 Go 客户端已能消费注入通道；核实 command-mode 的结构化输入/输出、日志持久化与跨沙箱身份/回执问题，提出一个有界 X1 试验。

写 docs/handoff-receipt.md：产品复述；真实模块/环境/基线命令与证据；设计代码冲突与假设；必须由我判断的少量事项及推荐代价；seed 保留/修改建议；建议先执行的一个任务及权限边界。

允许生成接管报告和获准检查所需的临时文件，不改业务代码和验收预期，不发布工单，不 commit/push/merge/release，不把任何标签当授权。完成回执后停止，等我批准下一步。
