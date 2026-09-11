# 任务系统适配

当前远端：未创建/未配置。不得猜测 owner/name，不自动 gh repo create。用户建立仓库后从真实 Git remote 核验；不把本包路径或建议名当远端。

GitHub 是正式任务发布后的唯一任务状态来源。Spec 是 Git 中的行为权威，Issue 链接批准 Spec 的 commit，不复制独立可变的第二份 Spec。

候选入口：`docs/issue-seeds/README.md`。稳定 Seed-ID 用于去重，不等于 Issue 编号。发布须有明确授权；先查看现有 Issue 和 Seed-ID，发布中断先查询再重试。发布后仅在 seed 补真实映射与历史，不继续维护任务状态。

发布授权、实现授权、本地提交授权、push/PR/merge/release 授权分别核对。标签不产生执行权。跨仓库 S3 不能凭本仓库授权改 Accord。

to-spec/to-tickets 输出先留本地草稿，不默认发布或贴 ready-for-agent。正式正文采用 Parent / What to build / Acceptance criteria / Blocked by / Out of scope / Evidence。已关闭依赖须证明真的交付，取消关闭不算满足。
