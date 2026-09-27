# Popeye 与 Headlamp 能力事实基线

核查日期：2026-09-26。本文只记录官方仓库、官方文档和官方 Release 可确认的能力，供 Teleskope 竞品对照使用；不把路线图或第三方插件宣传当成核心产品现状。版本锚点为 Popeye `v0.22.1`（2025-01-28）和 Headlamp `v0.45.0`（2026-08-20）。[Popeye releases](https://github.com/derailed/popeye/releases) · [Headlamp releases](https://github.com/kubernetes-sigs/headlamp/releases)

## Popeye：集群现状的只读 linter

- 直接读取**已部署的 live cluster**，不是扫描磁盘上的 YAML；可按当前/指定 kubeconfig context、单 namespace、全 namespace 或指定 sanitizer（linter）运行，也能以容器、集群内一次性 Job 或 CronJob 执行。它明确声明只读，不修改 Kubernetes 资源；示例 RBAC 主要是 `get/list`。[README: purpose, CLI, in-cluster and RBAC](https://github.com/derailed/popeye#popeye-kubernetes-live-cluster-linter)
- sanitizer 覆盖一组**内置且有限的 curated resources**，包括 Node、Pod、Service、Secret、ConfigMap、Deployment/StatefulSet/DaemonSet、PV/PVC、HPA、PDB、RBAC、Ingress、NetworkPolicy、Job/CronJob，以及 GatewayClass/Gateway/HTTPRoute；检查侧重状态、未使用资源、端口/selector、probe、request/limit、镜像标签、RBAC 和资源利用率等。官方同时说明“目前只看给定的一组资源”，不能等同任意 CRD 的通用分析器。[README: linters](https://github.com/derailed/popeye#linters)
- `spinach.yaml` 是策略调优层：可设 CPU/内存利用率阈值、按 FQN/正则/label/annotation 排除资源、按 code 排除问题、覆盖 code severity、限定镜像 registry。它增强的是 linter 配置，不是交互式修复或策略执行。[README: SpinachYAML](https://github.com/derailed/popeye#spinachyaml)
- 报告提供 standard、无颜色 jurassic、YAML、HTML、JSON、JUnit、Prometheus 和单值 `score`（0–100）；可落盘、上传 S3/MinIO，或推送 severity/code/linter/error/cluster-score 指标到 Prometheus Pushgateway，并附示例 Grafana dashboard。[README: output formats and Prometheus](https://github.com/derailed/popeye#output-formats)
- 评分是各资源组问题汇总后的 Popeye Score，并以 OK/Info/Warn/Error 分级；官方 README 没把它描述为安全证明、升级证明或修复闭环。缺少 metrics-server 时，资源过/低配与容量类判断会少一层证据；RBAC 不足也会限制扫描。[README: report morphology and known issues](https://github.com/derailed/popeye#report-morphology)

## Headlamp：RBAC 驱动的读写 Kubernetes UI

- 核心定位是通用 Kubernetes Web UI：浏览列表和详情，同时提供有权限约束的创建、更新、删除，且操作可取消；内置资源编辑器、Pod 日志和 exec/terminal。删除/更新控件会按用户 RBAC 隐藏或禁用，因此它是管理/排障面，不是只读报告器。[official README: features](https://github.com/kubernetes-sigs/headlamp#features)
- 官方资料可确认的资源体验包括 Pod、Service、ConfigMap、Deployment、StatefulSet、DaemonSet、ReplicaSet、Job、CronJob、volume、Gateway API 等列表/详情，以及 Events、搜索、Resource Map 和 application-oriented Projects。`v0.45.0` 又加入 scheduling、Gateway API 视图和更多结构化创建表单；但“任意资源都有同等深度的专用页面/关系图”不应由此推出。[v0.45.0 release](https://github.com/kubernetes-sigs/headlamp/releases/tag/v0.45.0) · [Projects overview](https://headlamp.dev/blog/2025/11/11/headlamp-projects/)
- 排障闭环强于静态报告：Pod 日志可实时查看并切换 container，Pod 可直接 exec；官方迁移指南把 logs、exec、metrics、events 作为 UI 内的核心调试循环。[official migration guide](https://headlamp.dev/blog/2026/05/15/kubernetes-dashboard-migration/)
- 多集群是核心能力；Desktop 可从一个或多个 kubeconfig 读取多个 context，in-cluster 也可配置一个或多个 kubeconfig。认证支持 bearer token、client certificate 与 OIDC，授权最终服从 Kubernetes RBAC；Headlamp 本身不提供内建用户名/密码认证。[README](https://github.com/kubernetes-sigs/headlamp#features) · [installation/auth](https://headlamp.dev/docs/latest/installation/) · [OIDC](https://headlamp.dev/docs/latest/installation/in-cluster/oidc/) · [basic-auth boundary](https://headlamp.dev/docs/latest/installation/in-cluster/basic-auth/)
- 交付形态包括 Linux/macOS/Windows Desktop 和集群内部署；官方 Helm chart、容器镜像和 ingress 示例构成 Web/集群部署路径。[desktop installation](https://headlamp.dev/docs/latest/installation/desktop/) · [in-cluster installation](https://headlamp.dev/docs/latest/installation/in-cluster/) · [Helm chart](https://github.com/kubernetes-sigs/headlamp/tree/main/charts/headlamp)
- 插件是正式扩展面，可增加 routes、sidebar、resource detail/list sections、actions、themes、settings、UI panels 和 project CRD tracking。官方插件库包含 Prometheus（workload 详情图表，要求集群已有 Prometheus）、OpenCost、cert-manager、Cluster API、KEDA 等；这类能力应标为插件/依赖能力，不应全部计入无配置核心。[plugin guide](https://headlamp.dev/docs/latest/development/plugins/) · [plugin functionality](https://headlamp.dev/docs/latest/development/plugins/functionality/) · [official plugins](https://github.com/headlamp-k8s/plugins)
- 指标有两层：核心 UI 能使用 Kubernetes Metrics API 展示资源用量；更丰富的 workload 时间序列图表来自官方 Prometheus 插件并依赖 Prometheus。官方插件清单明确了这个依赖边界。[official plugins: Prometheus](https://github.com/headlamp-k8s/plugins#current-plugins)

## 用于差异评估的边界

Popeye 的直接对照轴应是“live lint 覆盖、规则可调性、严重度/分数、机器可读与 CI/定时输出”；Headlamp 的直接对照轴应是“通用资源管理、实时排障、RBAC/认证、多集群、插件和部署形态”。二者解决的问题不同：Popeye 不提供资源管理 UI，Headlamp 的核心卖点也不是确定性的合规/迁移评分。比较时应分别标注核心、可选插件和外部依赖，避免把 Headlamp 插件生态或 Popeye 的 score 扩张成产品未承诺的能力。
