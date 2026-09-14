# OctoBus 接缝

## 上游事实与本项目提案分开

上游 OctoBus 的模型是 service → instance → capset/method binding。已选择的 unary 方法可用 Connect RPC / MCP / gRPC。本文采用其 README 演示的 Connect unary 路径形状：

```text
/capsets/<capset>/connect/<instance>/<package.Service>/<Method>
```

本项目定义 `compliance.v1.KnowledgeService/Search`，契约位于 `protocol/compliance.proto`。这不是 OctoBus 内置能力。`experiments/x1/service/` 的合成 package 已在 X1 中真实导入并调用；真实知识系统仍须按来源、授权和实际接口另行批准建设。

## 最小外部能力

一个方法即可启动：Search。输入 query/topic/asOfDate/jurisdiction/industry/limit；输出 corpus + truncated。每条来源带 documentId、version、locator、text、sha256、日期、范围、来源类别和核验记录。

知识服务必须返回当前查询完整的相关候选，包括潜在冲突版本；如果因 limit 不能完整返回就 `truncated:true`。智能体收到后拒绝形成答案，不在第一批结果中任选版本。M0 最多接受 2000 个来源、4 MiB 回包；此限制不构成大规模检索性能承诺。

外部服务仍须实施基于真实主体/组织的权限过滤；请求的 industry 是查询范围，不是身份。不要将未经授权客户文本返回给 Agent，再指望提示词隔离。

## 两条路线

**Direct Connect**：`octobus_connect` 模式由受信配置固定完整 endpoint 和 token 引用，仅调用 Search，发送 Connect-Protocol-Version: 1；目前仅做模拟协议验证。该 token 是网关/外层认证配置，不凭字段名推断细粒度权限；远程认证与只读 ACL 仍需真实验证。

**X1 原生候选**：`octobus_native_x1` 已通过 agent-compose 的 `octobus_servers + qualified capset_ids`，使用 guest 的 `CAP_GRPC_TARGET/CAP_TOKEN` 调用真实合成 Search。固定 grpcurl/proto、sandbox 凭据内部展开，上游 token 留在 daemon，不自动回退 Direct。guide 注入失败可能只告警，所以 sandbox started 不是查询成功证明。

接入或部署原生路径前，必须读取 [X1 执行记录与 ADR](../../docs/specs/x1.md)：该候选仅验证 synthetic/extractive，plaintext 专用网络、daemon 管理级测试凭据、跨 runtime 回执丢失和取消 UNKNOWN 都有明确边界；并非正式生产 Adapter。

## 请求示意

```json
{
  "schemaVersion": "cqa.search/v1",
  "query": "演示材料中的资产清单需要记录什么？",
  "topic": "mlps",
  "asOfDate": "2026-09-10",
  "jurisdiction": "CN",
  "industry": "telecom",
  "limit": 100
}
```

响应形状见 protocol/compliance.proto；Direct Connect 的 httptest 与 X1 的真实原生 gRPC 试验是两套证据。后者已启动并调用 OctoBus，但仍未连接真实企业知识库。
