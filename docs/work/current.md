# 当前工作入口

当前入口：S1 r0 文档和三个工单已发布为 #1、#2、#3，#3 原生 blocked-by #1、#2。用户随后授权 S1-02 实施，改选 OMP 百智云 `grok-4.6`、当前仅记录 Token，并授权提交／推送候选、回写及关闭 #2。规格入口为 `docs/specs/s1.md` r1；实际发布版本和任务状态以 GitHub 回执为准，不在本文件维护第二套状态。来源／许可／人工答案与真实运行授权仍未解除。

S1 工单历史入口为 `docs/issue-seeds/S1.md`，新候选和证据入口为 `docs/development.md#s1-02-token-用量候选` 与私有 `.local/s1-02-token-usage/`；旧 `.local/s1-02/` 的费用证明 BLOCKED 保留。本地未提交的 X1 代码、执行文档和私有证据保持既有状态，不把 S1 模拟验证称作 X1 新候选。

继续 S1 时读取 `docs/specs/s1.md`、`docs/issue-seeds/S1.md`、`docs/knowledge-governance.md`、`docs/decisions-and-assumptions.md`，沿用 X1 的 S4 回执生存边界、S5 UNKNOWN 和 ADR-X1-01 限定风险。安全售前／咨询、全国等保定级备案、本地审核资料快照和另外四域边界不变，正式原生接入留 S2。生成客户端固定 `baizhi-chat/grok-4.6`；不嵌入 OMP 编码代理，不复制其凭据。当前仅记录 Token，不作费用估算或额度控制；成功重放不重复调用，失败／未知不重试。

当前模型／用量事实见 `docs/research/s1-baizhi-grok.md`；旧 `s1-model-candidate.md` 为历史。百智云的模型可用性、接口兼容、地域与条款尚未真实核验，不沿用旧百炼批准。来源研究仍见 `docs/research/s1-source-candidates.md`；代理转录和地方引用不证明2026现行效力，来源资格、许可、问法／日期和人工答案须审核。真实调用 NOT_RUN，正常样例来源前提 BLOCKED；新候选及最终运行清单固定后另行授权。

后续 OctoBus 对接 RAG 增强查询与自主材料上传已记录为演进意向，未纳入 S1。建议知识服务负责上传／审核／入库与权限，查询智能体仍只读；届时另定上传安全、客户数据及资料许可规格，不提前增加平台组件。

当前授权包括 S1-02 软件实施、接口资料研究及其 commit／push／#2 回写关闭，不包括其他工单实施、真实模型／业务服务调用、恢复 X1、客户资料或 merge／release。既有 X1 工作与冻结证据保留，不夹带进本次提交；原 S1-02 私有证据不改写。
