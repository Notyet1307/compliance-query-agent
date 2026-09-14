# 候选工单入口

本目录同时保留未发布候选与已发布历史种子，发布映射见各 seed，正式任务状态只在 GitHub 维护。候选和标签均不授予执行资格。推荐里程碑依赖：

```text
SEED-B0 → SEED-X1 → SEED-S1 → SEED-S2 → SEED-S3 → SEED-R0
```

B0 是接管检查，不包装成功能。X1 只验证一个关键运行接缝。S1/S2/S3 都以可演示业务闭环拆分；每张票自己完成必要验证，R0 只负责小里程碑的独立验收。

每个文件的 Seed-ID 是稳定去重标识，不是 GitHub 编号。用户批准发布后由 OMP 基于真实 remote/已有 Issue 复核；保留 GitHub 映射而不继续在此同步状态。禁止预建整个愿景任务树。

S2 的 [已审阅规格](../specs/s2.md) 已形成 [两张纵向切片及验收归属](S2.md#本轮候选拆分)，并在本次明确授权下发布。这里只保留真实编号和依赖，不继续同步状态；当前查询／草稿接受、延期重放及历史失败均以正式工单为准。

| Seed-ID | 正式工单 | 原生前置依赖 |
|---|---|---|
| SEED-S2-01 | [#5：仅查询权限的受管单文档检索](https://github.com/Notyet1307/compliance-query-agent/issues/5) | 无 |
| SEED-S2-02 | [#6：受管模型草稿与持久重放](https://github.com/Notyet1307/compliance-query-agent/issues/6) | #5 |

固定规格与候选提交为 `40115a8c8007d3a8e5febd47827e7ca077ab6028`，发布 PR 为 [#4](https://github.com/Notyet1307/compliance-query-agent/pull/4)。未发布父 S2 或后续愿景工单；工单、标签和本次 Git 发布均不授予新的模型／凭据／资源操作。
