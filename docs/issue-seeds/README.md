# 候选工单入口

所有条目均为 CANDIDATE，未发布、未获执行资格。推荐依赖：

```text
SEED-B0 → SEED-X1 → SEED-S1 → SEED-S2 → SEED-S3 → SEED-R0
```

B0 是接管检查，不包装成功能。X1 只验证一个关键运行接缝。S1/S2/S3 都以可演示业务闭环拆分；每张票自己完成必要验证，R0 只负责小里程碑的独立验收。

每个文件的 Seed-ID 是稳定去重标识，不是 GitHub 编号。用户批准发布后由 OMP 基于真实 remote/已有 Issue 复核；保留 GitHub 映射而不继续在此同步状态。禁止预建整个愿景任务树。
