# Matt Skills 与 OMP 的使用约定

本包提供仓库适配，不安装、不复制 Skills，也不宣称你的本机已经安装。

首次核对 setup-matt-pocock-skills 的当前文件/依赖与生效指令，复用现有 docs/agents 配置。这里只配置 issue-tracker、domain、skill-usage；**未选择 triage，因此不创建 triage-labels.md 或假标签体系**。

按需用：设计歧义 → grill-with-docs 及其依赖；术语 → domain-modeling；批准内容整理 → to-spec；当前里程碑切片 → to-tickets；实现 → tdd/可选 implement；定位难题 → diagnosing-bugs；审查 → code-review 或等价受控路径。

所有 Skill 的默认动作受本仓库 AGENTS 和本轮授权覆盖。to-spec/to-tickets 不自动发布/ready；implement 不自动 commit；不为了长模板扩范围。仅支持本机真实存在的 Skill 读取方式，不模拟另一 Harness 的工具。

OMP 是当前开发协作工具；agent-compose 的 provider pi 是产品运行配置中的概念，两者不是同一个会话，也不自动共享个人配置或记忆。
