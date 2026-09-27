# Weave Scope 产品、架构与交互事实基线

核查日期：2026-09-26。本文只使用 Weaveworks 归档的官方仓库、源码文档和 Release/CHANGELOG；研究锚点为最终版 `v1.13.2`（2021-04-09）。Scope 已明确标记为不再维护，因此本文描述的是可继承的产品设计，不把它当作当前 Kubernetes 兼容性基准。[官方 README](https://github.com/weaveworks/scope) · [v1.13.2 release](https://github.com/weaveworks/scope/releases/tag/v1.13.2) · [CHANGELOG](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md)

## 一句话结论

Scope 真正值得 Teleskope “reborn”的不是一张力导向图，而是一套**实时、对象可下钻、以关系和上下文为中心的排障工作台**：从应用级聚合一路钻到 Pod、容器、进程和主机，同时把连接、指标、元数据和有限控制放在同一个上下文中。Teleskope 应继承这条交互主线，但把数据边界改成 Kubernetes/EKS 原生、只读证据优先。Teleskope 不实现特权主机 Probe；运行时连接只通过明确配置的外部 evidence provider 接入，也不复制 Scope 的容器生命周期控制和过时运行时假设。

## 产品意图与系统边界

- 官方定位是为 Docker/Kubernetes 自动生成应用地图，用于“理解、监控和控制”容器化微服务；它不是通用 YAML 资源管理器，也不是离线 linter。[README: product intent](https://github.com/weaveworks/scope#deprecated-weave-scope---troubleshooting--monitoring-for-docker--kubernetes)
- 核心架构是 **Probe + App**。Probe 在每台主机采集 `/proc`、容器运行时、Kubernetes 元数据、网络连接和指标，生成 report 并主动推送给 App；App 合并 report、构造/渲染 topology，通过 WebSocket/HTTP 服务 UI 与控制请求。0.6.0 Release 明确记录了从 App 拉取转为 Probe 推送，以及 `--no-app` / `--no-probe` 的独立运行边界。[CHANGELOG: 0.6.0](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md#release-060) · [probe source](https://github.com/weaveworks/scope/tree/master/probe) · [app source](https://github.com/weaveworks/scope/tree/master/app)
- Kubernetes 部署体现了这个边界：一个集中 App Deployment，Probe 作为每节点 DaemonSet；后者需要 `hostNetwork`、`hostPID`、Docker socket、内核 debugfs 等高权限主机访问。这是 Scope 得到进程/连接可见性的代价，不应被误当成现代 Kubernetes UI 的默认权限模型。[v1.13.2 Kubernetes manifest](https://github.com/weaveworks/scope/releases/download/v1.13.2/k8s-scope.yaml)
- Report 不是截图，而是可合并的图数据模型。`Topology` 包含 nodes、controls、metadata/metric/table templates；`Node` 包含 adjacency、metadata、metrics、parents 和 children。边是有向的，并嵌在源 Node 的 adjacency 中。[report package v1.13.2](https://pkg.go.dev/github.com/weaveworks/scope@v1.13.2/report#Topology) · [report source](https://github.com/weaveworks/scope/blob/master/report/report.go) · [node source](https://github.com/weaveworks/scope/blob/master/report/node.go)

## Scope 如何组织图，而不只是“画图”

### 视图层级与实体

Scope 为同一事实提供多种语义投影，而不是把所有对象塞进一张图：

- 基础层有 Processes、Containers、Hosts；容器还可按 image、host、DNS/name 分组。
- Kubernetes 层逐步加入 Pods、Services、Deployments、ReplicaSets、DaemonSets、StatefulSets、CronJobs 与 Namespaces；非 Kubernetes 容器以 `Unmanaged` 节点保留，而不是从图上消失。
- 节点的 parent/children 关系支撑从聚合节点下钻到成员；adjacency 表示实际通信关系。聚合、连接和所有权是不同关系，Scope 没把它们压成一种边。

这些能力可由 0.9.0 的 Pod/Service、0.15.0 的 Deployment/ReplicaSet/namespace、1.4.0 的 DaemonSet，以及后续 StatefulSet/CronJob 插件控制记录交叉确认。渲染层再通过 map/reduce、filter、join 和 aggregate，把底层 connection 投影为不同语义层的图；聚合成员保留在 children 中。[CHANGELOG](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md) · [render pipeline](https://github.com/weaveworks/scope/blob/master/render/render.go)

### 分组、过滤、搜索

- topology selector 决定观察层级；namespace filter 和 system/application/both 三态过滤负责削减噪声，过滤状态会在相关子视图间保持。
- 0.15.0 引入 smart search：既能匹配名称、IP 和 metadata，也支持类似 `CPU > 50%` 的指标比较；搜索结果直接过滤当前图，而不是跳到独立结果页。
- 0.17.0 增加 Table Mode，明确用于图中节点过多时提高信息密度。Scope 因而承认：图不是所有规模下的唯一答案。
- UI 把 view state 放入 URL/localStorage，支持深链接、刷新后恢复、每 topology 保留 pan/zoom；这让“分享当前排障上下文”成为产品能力，而不是纯视觉状态。

来源：[CHANGELOG: search、Kubernetes filters、table mode、view state](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md) · [search component](https://github.com/weaveworks/scope/blob/master/client/app/scripts/components/search.js) · [search selectors](https://github.com/weaveworks/scope/blob/master/client/app/scripts/selectors/search.js) · [node filters](https://github.com/weaveworks/scope/blob/master/client/app/scripts/selectors/node-filters.js)

### 图上交互与详情面板

- hover/selection 高亮节点和相连边；点击节点打开 contextual details，而点击详情里的 parent/child/connection 又可切换到相应 topology 或对象。
- 详情面板按重要程度展示 metadata，再按需展开完整信息；包含 tags/labels、地址、状态、uptime/restart count、容器 image/ports/entrypoint、进程 command line/threads/file descriptors 等。
- children 和 connections 使用可展开、可排序的表格，并能从 host/service 聚合层寻找 CPU/内存最高的具体成员。README 把这种跨层深链列为核心卖点。[README: contextual details](https://github.com/weaveworks/scope#contextual-details-and-deep-linking)
- 指标既在详情中以 sparkline 呈现，也可在 canvas 节点上显示；0.14.0 的实现记录为 CPU/内存节点指标和每秒更新、60 秒历史的 sparkline。后续还有 resource-usage view。[CHANGELOG: 0.14.0 and 1.3.0](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md)
- 图支持方向箭头、暂停实时刷新、force relayout、高对比模式，以及当前 node graph 的 SVG 下载；0.15.0 另加入完整 report JSON 下载。[CHANGELOG: 0.13.0, 0.15.0, 1.3.0](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md)

## 实时排障与控制

Scope 把观察和操作放在同一上下文中：容器可 start/stop/restart/pause，host/container 可开 terminal，Pod 可看多容器日志或删除，Deployment/ReplicaSet/ReplicationController 可 scale。官方同时明确警告：启用这些能力后不能把 4040 暴露给不可信用户。[README: controls](https://github.com/weaveworks/scope#interact-with-and-manage-containers) · [CHANGELOG: 0.10.0 and 0.15.0](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md)

这说明 Scope 的成功交互不是“能修改所有 YAML”，而是**从异常节点直接触发少量高频排障动作**。但其授权模型和容器运行时控制面不适合原样复刻；Teleskope 若保持 evidence-first，应把动作定义为只读日志/事件/命令建议或跳转，写操作另设明确安全边界。

## 插件与导出

- Probe plugin 是本机 HTTP 插件，可为 host/container/process 等对象添加新 metrics、metadata、tables 和 controls；后续扩展到 HTTP links 和更多 Kubernetes 对象。插件影响的是图中现有对象的上下文，而不是任意前端页面框架。[README: plugins](https://github.com/weaveworks/scope#extend-and-customize-via-plugins) · [官方 plugin 文档](https://github.com/weaveworks/scope/blob/master/site/plugins.md) · [CHANGELOG: 0.14.0, 1.0.0, 1.9.0](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md)
- 导出有两类：当前可视 node graph 的 SVG，以及完整 report JSON。它们分别服务“分享所见”和“保留原始事实”，值得在 Teleskope 中继续分开。[CHANGELOG](https://github.com/weaveworks/scope/blob/master/CHANGELOG.md)

## UI 优点与已暴露的限制

### 值得保留

1. **先选语义层级，再看关系**：Service/Deployment/Pod/Container/Process/Host 是同一系统的不同投影，而不是一张万能图。
2. **图、表、详情同源**：图负责发现关系，table 负责密度，详情负责证据；三者共享过滤和 selection。
3. **渐进式下钻**：聚合节点显示总览与告警，children/connection 表可定位具体实例，并可深链回图。
4. **实时但不抖动**：保留 pan/zoom、selection 和 layout，提供 pause/force-layout，避免每次刷新都打断认知位置。
5. **指标贴着实体**：短期趋势、状态和连接直接出现在对象上下文中，不要求先切换到独立监控产品。
6. **规模退化路径明确**：节点过多时切 table、按 namespace/system 分类过滤，而不是无限缩小节点。

### 不应原样复刻

1. **特权主机 Probe**：`hostPID`/Docker socket/debugfs 带来巨大权限面，也绑定了 Docker 与旧内核采集方式。Teleskope 明确不实现或发布 node probe。核心使用 Kubernetes watch 与 Metrics API；运行时连接证据只能来自外部 CNI observability、OpenTelemetry 或 AWS network telemetry provider，并公开来源、窗口、采样和 coverage。
2. **一图混合事实和推断**：Scope 的通信连接、Kubernetes ownership、分组聚合与伪节点视觉相近，密集时容易误读；Teleskope 应显式区分 edge kind、source、freshness 和 confidence。
3. **力导向图作为默认大规模布局**：Scope 自己用 Table Mode、graph complexity check、force relayout、过滤和 node-limit 提示补救。Teleskope 应保留语义 swimlane/层级布局，并把力导向关系图作为可选探索模式。
4. **内建高风险控制**：terminal、删除 Pod、scale、容器 lifecycle 需要成熟认证/RBAC/审计；这不是 Popeye 替代路线的前置条件。
5. **60 秒内存指标等同监控**：短 sparkline 适合排障上下文，不替代 Prometheus 的长期查询、告警和容量分析。
6. **运行时/对象版本冻结**：最终版本面向旧 Kubernetes API 与 Docker-era 节点模型；应继承交互思想，不继承资源清单和采集实现。

## 对 Teleskope 的 Issue 拆分输入

以下不是最终 Issue 文案，而是从 Scope 事实提炼出的独立能力块：

1. **统一 live graph contract**：node/edge kind、parent/child、observed connection、source、timestamp、coverage；snapshot 和 live 使用同一结构。
2. **多语义 topology views**：Application/Service、Workload、Pod、Node、Network/EKS infrastructure；不同视图明确聚合规则。
3. **图—表—详情联动状态**：selection、hover、filter、search、namespace、URL deep link 和刷新状态保持。
4. **Scope 式 contextual drawer**：summary、findings、metrics、children、connections、metadata、events 统一分区，并能跨层下钻。
5. **smart search/query**：名称/label/IP/ARN/metadata 查询，加状态、finding severity、CPU/memory 等结构化比较。
6. **实时短窗指标 provider**：Metrics API 为基线，Prometheus/OTel 为可选；显示来源和时间范围，不伪装为长期监控。
7. **大图降级策略**：聚合阈值、table mode、邻域 focus、隐藏系统对象、复杂度提示和确定性布局。
8. **连接证据 provider**：建立只接外部 telemetry 的 provider seam；缺少流量证据时不能用声明式 Service selector 冒充实时连接，也不以自建特权 node probe 补齐。
9. **分享与导出**：可复现 URL state、SVG/PNG 当前视图、JSON evidence bundle 各自独立。
10. **只读排障 actions**：日志、事件、复制 `kubectl`、打开 AWS/Kubernetes 目标；任何写操作作为后续独立安全项目。
11. **上下文扩展点**：允许 provider 向既有对象追加指标、证据表、链接和只读动作；暂不建设任意 UI 插件平台。

## 推荐的继承边界

Teleskope 可把产品表达收敛为：**Popeye 的确定性健康检查与 EKS 特化是判断引擎；Scope 的实时关系探索与上下文下钻是交互外壳。** 两者应共享同一份对象身份、证据、coverage 与 finding，而不是各建一套 dashboard 数据模型。Teleskope 不建设特权 node probe；外部 provider 缺失时，产品仍完整呈现声明式与解析式关系，并将运行时连接覆盖标为 unavailable。第一阶段不需要 terminal、CRUD、容器 lifecycle 或通用插件商店；必须先证明从 finding/搜索结果进入 topology，再从聚合对象稳定下钻到证据对象的完整闭环。
