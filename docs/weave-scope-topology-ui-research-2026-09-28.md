# Weave Scope Topology 呈现与 Teleskope Mock 设计输入

核查日期：2026-09-28。研究固定在 Weave Scope 最终版 `v1.13.2`，只使用 Weaveworks 官方仓库内的产品文档、源码、CHANGELOG 和截图。本文只提供下一轮 mock 的事实基线和设计约束，不实现 mock。

## 结论先行

Scope 的核心并不是“把所有资源放进多层卡片”，而是以下组合：

1. **一次只选一种 topology projection**：Processes、Containers、Pods/Controllers/Services 或 Hosts。
2. **当前 projection 是一张有向关系图**：使用 Dagre 对连接关系做分层排布，客户端在上、服务端在下，不是 force-directed/free graph。
3. **节点本身极度节制**：形状、颜色、短名称、可选的一个指标；完整 metadata、metrics、connections、children 都在选中后的详情面板。
4. **选中是排障动作，不是普通高亮**：选中节点移到视口中心，一跳邻居围绕它，其他节点和边弱化，右侧同时打开 contextual details。
5. **Graph 和 Table 是并列呈现**：Scope 提供 Table Mode，并用复杂度信号决定初始模式和动画；该信号不是布局引擎的硬节点上限。0.14.0 已明确移除早期 100-node rendering limit。

因此，Teleskope 的下一个 mock 应删除 swimlane 和用户无法理解的 `Semantic view`，改成 **Scope-like projection selector + top-to-bottom relationship graph + focus/details**。“对外网络 → 工作负载 → 基础设施”可作为选中对象后的垂直排序约束，不应再画成三个永久横向框。

## 重要事实校正

### Scope 主图不是力导向图

官方 Feature Overview 明确说明节点以“clients above servers”的特定顺序呈现，图从上到下阅读；边代表节点间的 TCP connections。[官方 Feature Overview](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L19-L23)

前端实现创建的是 `dagre.graphlib.Graph`，将有向 adjacency 交给 `dagre.layout()` 计算节点和边的坐标。这是有向分层图布局，不是粒子之间不断迭代的 force simulation。[布局引擎源码](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L153-L220) · [Dagre graph 与 layout cache](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L422-L495)

Scope 底部的 “Force re-layout” 是“强制重算布局”，不是“使用 force-directed layout”；按钮文案也只说它可能减少边交叉，但会移动节点。[官方 footer 实现](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/footer.js#L19-L39)

> 校正影响：`docs/weave-scope-product-ui-research-2026-09-26.md` 中将 Scope 主布局概括为“力导向图”不准确。本文对 topology 呈现的描述以 Dagre 源码和官方 Feature Overview 为准。

## 已核实的 Scope topology 交互

### 1. 布局、节点与边

- **主布局**：当前 topology 的节点和有向边被送入一个 Dagre graph；有连接的节点由 Dagre 定位，没有连接的节点则另外排成网格。[布局引擎](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L72-L147) · [Dagre 运行段](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L153-L250)
- **方向**：官方文档定义为客户端在上、服务端在下；CHANGELOG 也记录了将边方向改为 client-to-server。[官方 Feature Overview](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L19-L23) · [CHANGELOG](https://github.com/weaveworks/scope/blob/v1.13.2/CHANGELOG.md#release-0100)
- **边的含义**：在主要 Processes/Containers/Pods/Hosts projection 中，边来自被观测的 TCP connections，而不是将 ownership、selector、scheduling 等关系都画成同一种线。Kubernetes Service/Controller 视图会把 Pod connection graph 映射到父对象，再聚合其连接。[官方 Feature Overview](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L19-L23) · [Pod/Service/Controller renderer](https://github.com/weaveworks/scope/blob/v1.13.2/render/pod.go#L43-L131)
- **外部网络**：Scope 不为 Internet 画固定区域，而是生成普通 pseudo node；代码将它拆成 incoming/outgoing 两个 ID，节点的 minor label 分别为 `Inbound connections` 和 `Outbound connections`。[官方 Internet node ID 与判定](https://github.com/weaveworks/scope/blob/v1.13.2/render/id.go#L9-L18) · [外部连接映射](https://github.com/weaveworks/scope/blob/v1.13.2/render/id.go#L79-L101) · [Internet label](https://github.com/weaveworks/scope/blob/v1.13.2/render/process.go#L9-L15)
- **形状编码**：Process 为方形、Container 为六边形、Host 为圆形、Pod/Service/Deployment 为七边形；其他 workload/storage 类型也有对应形状。[官方 report 默认形状](https://github.com/weaveworks/scope/blob/v1.13.2/report/report.go#L225-L316)
- **实时稳定性**：每个 topology/options 组合有独立布局 cache；无新节点时直接沿用原坐标，少量新节点尽量插入现有 rank，只有变化较大才全量重排。[布局 cache 和增量策略](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L422-L495)

官方 [Graph Mode 截图](https://github.com/weaveworks/scope/blob/v1.13.2/site/images/topology-map.png) 也展示了这种表达：小型形状 glyph、形状外部的短 label、大量留白和细连接线，而不是承载多行 metadata 的大卡片。

### 2. Processes / Containers / Pods / Hosts 如何切换

- 官方产品把 Views 解释为对容器化系统的高层过滤器，分为 Processes、Containers、Orchestrators 和 Hosts；Kubernetes 存在时，Orchestrator 才会出现 Pods、ReplicaSets、Deployments 和 Services。[官方 Views 说明](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L25-L39)
- 实现上，主 selector 依次注册 Processes、Containers、Pods 和 Hosts；Processes 有 `by name`，Containers 有 `by DNS name`/`by image`，Pods 下有 `Controllers`/`Services`，Hosts 下有 `Weave Net`。[官方 topology registry](https://github.com/weaveworks/scope/blob/v1.13.2/app/api_topologies.go#L204-L304)
- 这些选项切换的是同一份 report 的不同 render projection，不是把 Processes、Containers、Pods 和 Hosts 同时堆到一张图的固定分层中。子 topology 是主 topology 下的 selector item，每次只有一个 `currentTopology`。[官方 selector 组件](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/topologies.js#L17-L94) · [topology request actions](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/actions/request-actions.js#L589-L605)
- Kubernetes namespace filter 被加到 Containers、Pods、Services 和 Controllers 及其子视图；Container 还有 application/system/all、running/stopped/both 和 show/hide uncontained 选项。[官方 topology options](https://github.com/weaveworks/scope/blob/v1.13.2/app/api_topologies.go#L115-L145) · [Container filters](https://github.com/weaveworks/scope/blob/v1.13.2/app/api_topologies.go#L160-L202)

### 3. 节点内容与右侧详情的边界

**图上节点只放：**

- 形状、类型色、主 label、可选的 minor label、聚合 stack 效果、可选的一个 metric 值/填充量，以及 search match 的少量附加提示。[官方 NodeContainer](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/node-container.js#L43-L87) · [Node 渲染入口](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-chart-elements.js#L151-L177)
- 官方截图中长名称显示为节点下方的截断文本，不把 label 撑大成多行 object card。[官方 Graph Mode 截图](https://github.com/weaveworks/scope/blob/v1.13.2/site/images/topology-map.png) · [CHANGELOG 的 node-name length 调整](https://github.com/weaveworks/scope/blob/v1.13.2/CHANGELOG.md#release-0130)

**选中后的详情面板才放：**

- 完整标题和 parent links、controls、Status metrics、Info metadata、inbound/outbound connections、children 表格、plugin tables。过长 table 数据会明确显示 truncation warning。[官方 NodeDetails](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/node-details.js#L177-L268)
- 详情中的 parents、connections 和 children 是可继续下钻/跳转的对象；官方产品文档强调的是从 container 内进程到所在 host 的 contextual drilldown。[官方 contextual details 说明](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L59-L63) · [官方详情截图](https://github.com/weaveworks/scope/blob/v1.13.2/site/images/contextual-details.png)

### 4. Search、filters、metrics、edges、zoom/pan 和 focus

- **Search** 能查名称、label/path、metadata 和指标比较，并支持叠加条件；实际提示示例包含名称、`ip:value` 和 `cpu > 2%`。[官方 Search 说明](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L53-L57) · [Search 组件](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/search.js#L58-L123)
- **Search 的图上反馈**是弱化未命中节点/边，而不是立即删除图结构；Table Mode 则直接只保留命中行。[图上 blur 逻辑](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-chart-elements.js#L97-L145) · [Table search filter](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-grid.js#L135-L175)
- **Filters** 是当前 projection 的明确参数，例如 namespace、system/application、running/stopped、unmanaged/uncontained，而不是另外一个含义不明的 view。[官方 filters 说明](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L47-L51) · [过滤实现](https://github.com/weaveworks/scope/blob/v1.13.2/app/api_topologies.go#L115-L202)
- **Metrics** 可作为图节点的单一 pinned metric，也会在详情面板的 Status 中展开；Table Mode 则强调像 `top` 一样根据资源消耗查看对象。[官方 Graphic/Table 说明](https://github.com/weaveworks/scope/blob/v1.13.2/site/features.md#L41-L45) · [Node metric 入口](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/node-container.js#L52-L80)
- **Hover** 会高亮节点及其正反向一跳邻居/边；hover 一条边会高亮两端节点。[官方 hover selectors](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/selectors/graph-view/decorators.js#L7-L68)
- **Select/focus** 会将选中节点固定在视口中心，把直接邻居排成一圈，重画对应直线边，并弱化不相关节点/边。[官方 focus layout](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/selectors/graph-view/layout.js#L49-L170) · [focus/blur 装饰器](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-chart-elements.js#L91-L149)
- **Edge direction** 不对所有边常驻画箭头；只有 focused 或 highlighted 边显示终点箭头，降低默认画面噪声。[官方 Edge 组件](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/edge.js#L35-L60)
- **Pan/zoom** 由 SVG drag 和 wheel 控制，并将缩放状态按 topology/layout 缓存；布局切换或强制 relayout 时恢复对应 zoom state。[官方 ZoomableCanvas](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/zoomable-canvas.js#L35-L127) · [CHANGELOG 的 per-topology pan/zoom](https://github.com/weaveworks/scope/blob/v1.13.2/CHANGELOG.md#release-0130)
- **大图复杂度**：当 `node_count + 2 * edge_count > 500` 时，代码将图判定为高复杂度，用于选择初始 Table Mode 并关闭动画；Dagre 布局函数本身没有这个硬停止条件。0.14.0 的 CHANGELOG 明确记录了早期 100-node rendering limit 已移除。[官方 complexity selector](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/selectors/topology.js#L42-L50) · [CHANGELOG 0.14.0](https://github.com/weaveworks/scope/blob/v1.13.2/CHANGELOG.md#release-0140) · [Table Mode release note](https://github.com/weaveworks/scope/blob/v1.13.2/CHANGELOG.md#release-0170)
- **开阔度与孤立对象**：Scope 让 Dagre 使用放大的节点占位、明确的 `nodesep`/`ranksep` 计算完整画布包围盒；0-degree 节点不塞进 directed rank，而是在连通图右侧或下方排成近似方形网格。布局 cache 在无新节点时复用坐标，少量新节点按已有 rank 插入；发现节点距离过近时再全量重排。[官方布局源码](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L72-L250) · [cache 与 overlap defense](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L422-L495)

### 5. Scope 没有做什么

以下是从官方实现可以直接证明的边界：

- **主 Graph Mode 没有固定 swimlane**：它对当前 projection 构建一个 Dagre graph，依赖边的方向生成 rank；没有按 resource kind 画永久的分区背景。[官方 graph selector](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/selectors/graph-view/graph.js#L15-L61) · [Dagre 布局](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/nodes-layout.js#L153-L220)
- **主 Graph Mode 没有同时显示“应用/运行时/基础设施”的 semantic cards**：Processes、Containers、Pods、Hosts 是相互切换的 projection，而不是并行展示的分层容器。[官方 topology registry](https://github.com/weaveworks/scope/blob/v1.13.2/app/api_topologies.go#L204-L304) · [官方 selector 组件](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/topologies.js#L17-L94)
- **图节点不是 object summary card**：GraphNode 接收的可视主体是 shape/label/minor label/stack/metric，丰富信息进右侧 details。[官方 NodeContainer](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/charts/node-container.js#L52-L87) · [官方 NodeDetails](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/node-details.js#L177-L268)
- **Scope 后期确实有单独的 `Resources` mode，但它不是主 topology 关系图**：它只对 Hosts → Containers → Processes 的 CPU/Memory 消耗做三层宽度表达，层次和指标都是硬编码；资源模式中 search 也不受支持。[官方 resource constants](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/constants/resources.js#L2-L27) · [resource layout](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/selectors/resource-view/layout.js#L30-L114) · [selector 对 resource search 的限制](https://github.com/weaveworks/scope/blob/v1.13.2/client/app/scripts/components/topologies.js#L34-L68)

因此，Scope 的 Resources mode 不应被解读为保留 Teleskope 当前 `Semantic view` 的理由。它是一个有限的资源占用分解图，不是系统拓扑的默认表达。

## 对 Teleskope 的设计建议

以下是建议，不是对 Scope 的事实描述。

### 信息架构

1. 将页面顶部的 `Semantic view` 和 swimlane mode 都移除。
2. 只保留用户能理解的 **projection selector**：`Workloads`、`Pods`、`Nodes`、`EKS / Network`。第一版 mock 可先只完整展示 `Workloads`，其他 selector 保留可见状态。
3. 同一 projection 中只展示对当前排障问题有用的节点粒度。不要在默认全局图同时铺开 Workload、Pod、Container、Node、Subnet 和 VPC。
4. 跨层关系通过 **选中后的 focus neighborhood + details drawer** 展示。这样可以在一次选中中表达“对外网络 → 工作负载 → 基础设施”，却不需要永久 swimlane。

### 默认图

- 使用从上到下的 directed graph，不使用力导向漂浮。默认 Workloads projection 中，优先以请求/调用关系定 rank；对缺少 observed traffic 的边显示明确的 evidence kind，不冒充 Scope 式 TCP connection。
- External/Internet 使用少量 pseudo node，放在图的上游/边缘；只有证据支持时才连到 Service/Ingress/LoadBalancer/Workload。
- Node/EKS infrastructure 不在 Workloads 全局图中永久占一个下层区域。选中 workload 或切到 `Nodes`/`EKS / Network` projection 时再呈现 placement 和 infrastructure chain。
- 边的 kind 必须可区分：`observed traffic`、`routes to/selects`、`owns`、`scheduled on`、`attached to`。默认低对比度，hover/select 时显示箭头、kind、source、freshness/coverage。

### 节点与溢出策略

- 节点不再是容纳多行资源信息的“object 框体”，而是 Scope 式紧凑 glyph + label。
- 图上只允许：类型 glyph、短名称、一个状态/finding 标记、一个可选 metric。其他信息进 drawer。
- label 容器必须有硬上限：最多两行，超出使用 ellipsis；全称通过 native tooltip/focus tooltip 和 drawer 标题可达。
- Namespace/kind 不和对象名称挤在一行：它们是 filter/context，选中后再在 drawer 中完整显示。
- 必须对长英文名、长 ARN、长 image digest、无断点字符串做专项样式验收，任何 zoom 下都不得撑破节点或覆盖邻居。

### Focus 与 details

- hover 只高亮直接邻居和连接边。
- click 进入 focus：选中节点居中，一跳邻居围绕它，其他内容弱化，右侧 drawer 打开。
- focus 内如果要表达层次，才使用松散的垂直 rank：External/entry 在上，当前 workload 在中，Pods/Nodes/EKS attachments 在下。不画 lane 背景，不要求每层必须填满。
- drawer 顺序建议为 `Summary`、`Findings`、`Metrics`、`Connections`、`Children / Placement`、`Evidence`，并支持从表格行继续切换 projection/聚焦目标。

### Search、filter 和规模呈现

- Search 保持图的心智地图：命中对象和一跳边高亮，未命中弱化；提供“仅显示匹配”的显式动作，不在输入时默默销毁图结构。
- Filter 使用可见 chips：namespace、kind、health/finding severity、system/application、evidence availability。
- Graph 不设置会强制改写 presentation 的节点/边阈值；画布按布局包围盒扩张，复杂度只影响动画和非必要装饰。Table Mode 保留为用户主动选择的密集证据视图。

### 与 Scope 不同但必须保留的 Teleskope 边界

- **不实现 node probe**。运行时 connection 只能来自明确配置的外部 evidence provider；缺失时显示 unavailable/coverage gap。
- Scope 的边默认可被理解为 TCP connection，Teleskope 不能这样简化。声明式、推导式、观测式边必须标明 source、kind、freshness 和 confidence/coverage。
- Scope 的 terminal/container lifecycle 控制不进入本次 mock；Teleskope 保持只读 evidence-first。

## Mock 验收清单

- [ ] 默认页面不出现 swimlane 背景或 `Semantic view` 文案。
- [ ] 顶部使用明确对象 projection selector，至少完整展示 `Workloads`。
- [ ] 默认图为 top-to-bottom directed relationship graph，节点位置稳定，不做持续力导向漂移。
- [ ] External/Internet 作为可选 pseudo node，无证据时不生成伪连接。
- [ ] 节点只有 glyph、受控 label、一个状态/finding 标记和一个可选 metric。
- [ ] 超长 Deployment/Service 名、ARN、image digest 不溢出、不改变节点尺寸、不覆盖邻居；全值可通过 tooltip 和 drawer 获取。
- [ ] hover 高亮一跳邻居和边；click 将对象居中、弱化其他对象并打开 drawer。
- [ ] focus 态可读地表达“对外网络 → 工作负载 → 基础设施”，但不使用固定 lane 框。
- [ ] drawer 容纳完整 metadata、findings、metrics、connections、children/placement 和 evidence，图节点不重复这些内容。
- [ ] 每种边都有可辨识的 kind；hover/select 能看到 direction、source 和 freshness/coverage。
- [ ] Search 高亮/弱化图上结果，filter 以可见 chips 显示，两者不需要另一个 semantic mode。
- [ ] Small/Medium/Large fixture 在用户选择 Graph 时都保持 Graph；复杂度不得自动切 Table，可通过开阔画布、聚合、LOD 和用户主动过滤降低噪声。
- [ ] pan/zoom 、selection 和当前 projection/filter 在打开/关闭 drawer 时保持稳定。
- [ ] mock 不出现 node probe、terminal、容器 lifecycle 或任何写操作。
