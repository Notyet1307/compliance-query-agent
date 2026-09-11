# OctoBus 接缝

## 上游事实与本项目提案分开

上游 OctoBus 的模型是 service → instance → capset/method binding。已选择的 unary 方法可用 Connect RPC / MCP / gRPC。本文采用其 README 演示的 Connect unary 路径形状：

```text
/capsets/<capset>/connect/<instance>/<package.Service>/<Method>
```

本项目提出 `compliance.v1.KnowledgeService/Search`，定义在 `protocol/compliance.proto`。**这不是 OctoBus 内置能力；启动包不包含可运行的知识服务 package，也未向 OctoBus 导入任何服务。** 真实知识系统/Node service package 由 S1/S2 按实际接口建设。

## 最小外部能力

一个方法即可启动：Search。输入 query/topic/asOfDate/jurisdiction/industry/limit；输出 corpus + truncated。每条来源带 documentId、version、locator、text、sha256、日期、范围、来源类别和核验记录。

知识服务必须返回当前查询完整的相关候选，包括潜在冲突版本；如果因 limit 不能完整返回就 `truncated:true`。智能体收到后拒绝形成答案，不在第一批结果中任选版本。M0 最多接受 2000 个来源、4 MiB 回包；此限制不构成大规模检索性能承诺。

外部服务仍须实施基于真实主体/组织的权限过滤；请求的 industry 是查询范围，不是身份。不要将未经授权客户文本返回给 Agent，再指望提示词隔离。

## 两条路线

**当前代码**：octobus_connect 模式由受信配置固定完整 Connect endpoint 和 token 引用，仅调用 Search，发送 Connect-Protocol-Version: 1。该 token 是所部署网关/外层认证方式的配置项，不凭字段名推断 OctoBus 天然提供细粒度身份权限。直接模式的远程认证与只读路径 ACL 需要真实验证。

**目标推荐**：agent-compose 原生 octobus_servers + qualified capset_ids，保留上游 token 在 daemon。上游说明原生能力网关配置/guide 注入可能在失败时只告警仍启动 sandbox，因此本业务必须另做硬性的查询可用性/权限验证，不能把 sandbox started 当数据链路成功。

原生注入的实际 guest 变量/代理调用协议尚未适配，本包不捏造名称。X1 只回答这一个运行接缝问题；根据结果实现正式 Adapter，或明确批准直接 Connect 的权限成本。

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

响应形状见 protocol/compliance.proto；本地 httptest 验证了客户端请求/解析，但并未启动 OctoBus，更未证明已连接实际企业知识库。
