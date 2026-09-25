# Product Hunt 发布就绪检查 · 2026-09-25

检查日期：2026-09-25（Asia/Shanghai）；远端仓库和公开 demo 检查于 02:51 UTC（北京时间 10:51）。远端事实通过 GitHub API、公开 URL 和 PH 官方文档实时核查；本地基线为 `f638c0b`，本轮 README highlights 已生成。此报告没有执行推送、部署或 PH 提交。

**结论：当前 NO-GO；可以继续准备发布素材，但不建议现在排期。** 已有可下载版本、六平台二进制 smoke 和合成示例，用户随后已确认 Apache-2.0，仓库已添加 LICENSE 和打包校验；主要缺口是公开可交互 demo、发布版本与演示一致性、完整数据处理说明及 PH 账户/素材确认。这是本项目的发布质量门槛，不代表 PH 强制要求所有项目都具备这些文档。

## 已核实与缺口

| 项目 | 状态 | 本轮证据与边界 |
| --- | --- | --- |
| 公开仓库 | 已核实 | [仓库](https://github.com/kuraudo-lab/teleskope) `isPrivate=false`；API 的 description 为空、homepage 为 null。补充一句描述和 demo 首页有助于新访客，但不是 PH 强制项。 |
| 稳定版下载 | 已核实 | [v0.15.0](https://github.com/kuraudo-lab/teleskope/releases/tag/v0.15.0) 于 2026-09-22 发布，非 draft、非 prerelease，含 Darwin/Linux/Windows × amd64/arm64 六个包与 `checksums.txt`。 |
| CI 与 release | 已核实，范围有限 | [CI](https://github.com/kuraudo-lab/teleskope/actions/runs/35734467118)、[Release](https://github.com/kuraudo-lab/teleskope/actions/runs/35734466995) 成功，对应 `2c75395`。六个原生 runner 的 smoke 均成功。检查包括 checksum、解包、`--version`、`--help`、缺失 kubeconfig 的错误输出；不是六平台公开安装脚本端到端验收。见 [workflow](../../.github/workflows/build.yml)、[smoke 实现](../../scripts/release.py)。 |
| 公开安装实测 | 未完成 | 在 macOS arm64 将公开 `main/scripts/install.sh` 下载到 `/private/tmp/teleskope-ph-audit-20260925`，与本地脚本 `cmp` 一致。首次执行返回 `curl: (52) Empty reply from server`；第二次返回 `curl: (56) Recv failure: Operation timed out`，进程已结束，没有遗留后台安装。单独下载 v0.15.0 的公开 `checksums.txt` 成功。此结果不能认定安装已通过，也不能把网络失败直接归因于产品。Linux/Windows 本轮未执行。 |
| 对外版本与演示一致 | 待完成 | API 查询的远端 main 仍为 `2c75395`；本地已有拓扑选择、详情抽屉、#31 合成 demo 和本轮素材更新。这些尚未包含在 v0.15.0 中。发布前选定一个版本，并用该版本重拍/核对素材。 |
| 许可证 | **本地已完成；待公开同步** | 用户在本次审计后确认 Apache-2.0。已添加 [官方 LICENSE 正文](../../LICENSE)、README 声明及发布包 LICENSE 校验。前述远端审计时 `license=null` 是历史结果；尚未推送，既有 v0.15.0 发布包未被修改。第三方依赖仍遵循各自许可，本次不代表完整依赖许可证审计。 |
| 零凭据示例 | 本地已备齐；公网缺口 | [demo guide](demo-assets.md) 与 [catalog](../demo/index.html) 覆盖 EKS/Gateway/Storage/Hub/AI。GitHub HTML 文件查看页不能直接运行交互 UI；[预期 Pages URL](https://kuraudo-lab.github.io/teleskope/demo/) 返回 HTTP 404，API `has_pages=false` 且 Pages API 404。本轮没有发现其他已配置公开 demo 地址，不推断一定不存在外部站点。 |
| 数据处理说明 | **P0：需集中补齐** | README 已说明 Secret 仅元数据、ConfigMap 可能含内容、默认 loopback、无内置 HTTP 认证、AI 显式操作、API key 不进入浏览器。缺少面向新用户的一页说明：原始报告/导出包含什么、凭据使用与保存、分享前检查、Hub 接收边界、AI 发送字段及供应商留存策略由用户自行配置。`docs/llm-analysis.md` 混有设计计划，不能将计划当现有保证。 |
| README highlights | 已生成 | [Topology](../../assets/highlights/topology.gif)、[Evidence](../../assets/highlights/evidence.gif)、[Fleet](../../assets/highlights/fleet.gif) 均为实际 UI 关键帧 walkthrough，每段 9 秒、1280×720，约 549/507/693 KiB；9 张原始 UI 截图保留在 `assets/highlights/frames/`。这是关键帧演示，不是连续录像；PH 专用画廊尚未制作，仍需验证尺寸、首帧与上传预览。合成数据与手工 AI 示例标记应保留。 |
| PH 专用素材 | 待验证 | 没有已核验的 PH 上传预览。需要至少两张可读画廊图、独立方形图标；README GIF 不等于已具备完整 PH 素材。视频是可选项，不单独作为阻塞。 |
| PH 账户与排期 | 所有者确认 | 未登录或检查个人 PH 账户、发布权限、maker usernames、历史 launch、排期。不能从仓库状态推断这些已就绪。 |
| 用户反馈 | 未验证 | 未发现可以确认完成的 5–10 位目标用户反馈记录；适合在正式排期前验证首次安装和 demo 理解成本。 |

## 当前 PH 官方规则

以下仅采用 PH 官方帮助中心和官方 launch guide；有冲突时保守准备，并以实际提交表单验证。

- 必须用个人账户发布；公司/品牌账户不能发布。新个人账户通常等一周，官方另列订阅 newsletter 可提前取得权限。账户实际是否获准仍需本人确认。[账户权限](https://help.producthunt.com/en/articles/481909-how-can-i-get-access-to-post)
- 画廊至少两张，推荐 `1270×760`；thumbnail 推荐 `240×240`，GIF thumbnail 小于 3 MB、悬停播放，首帧应能独立表达产品。不要把“README GIF 可播放”当成“PH 会自动播放”。[发布表单说明](https://help.producthunt.com/en/articles/479557-how-to-post-a-product)
- 官方 launch guide 接受画廊 GIF，并在 thumbnail 段落写所有图片需小于 3 MB；帮助中心把 3 MB 明确写在 thumbnail GIF 上。因此本项目采取保守导出目标：PH 图片/GIF 均小于 3 MB，再验实际上传结果；这不是声称已证明所有 gallery 类型存在相同硬限制。[素材清单](https://www.producthunt.com/launch/preparing-for-launch)
- Tagline 最多 60 字符。Description 官方 guide 写 500、帮助中心写 260，先准备不超过 260 的稿件可兼容二者。主链接可用 GitHub repo；自建 landing page 不是 PH 强制要求。视频可选，只接受完整且非 private 的 YouTube URL。[guide](https://www.producthunt.com/launch/preparing-for-launch)、[表单](https://help.producthunt.com/en/articles/479557-how-to-post-a-product)
- 当前流程为 Create Draft 或 Schedule，支持未来 30 天内择日。旧 guide 仍出现 Launch now，应以专项帮助说明为准。[调度](https://help.producthunt.com/en/articles/2724119-how-to-schedule-a-post)、[Launch Now 已替换](https://help.producthunt.com/en/articles/9823193-where-did-launch-now-go)
- 官方按 Pacific 时间日界线运行，并将默认上线写为 12:01 AM PST。排期时核对 UI 时区与当地夏令时，不机械把文档中的 PST 换算成固定北京时间。Maker 首评应请求真实反馈，不索取 upvote。[发布说明](https://help.producthunt.com/en/articles/479557-how-to-post-a-product)、[首评建议](https://www.producthunt.com/launch/preparing-for-launch)

可直接用于表单的短稿（定位仍待所有者认可）：

> Tagline: Understand your Kubernetes clusters before you change them.
>
> Description: Inspect EKS and Kubernetes inventory, topology, and migration gaps in local reports. Share evidence without granting cluster access. Try a synthetic demo; optional AI analysis stays separate from deterministic findings.

Tagline 为 59 字符，description 为 219 字符。上面的文案避免把扫描结果宣传为流量健康、备份可恢复或升级安全保证，文案保留了审计时的措辞；Apache-2.0 已在本地落地，公开同步后可采用 Open Source 定位。

## 到 GO 的最短路径

1. **P0，所有者决定：** 首发版本/成熟度、主要 CTA、PH 个人账户与 maker、发布时间。建议 CTA 优先公开 demo，其次下载；AI 保持可选能力，手工输出示例不能当成模型实测。
2. **P0，交付：** 让无 Kubernetes/AWS 凭据的访客点开可交互合成 demo；公开链接可达且内部链接/下载正常。部署方式可为 Pages 或其他静态托管，不要求另造复杂官网。
3. **P0，交付：** 补集中数据处理说明并从 README 链接；同步公开已采用的 Apache-2.0 LICENSE。发布选定版本，使图像、demo 与下载版本一致，CI 在该版本成功。
4. **P0，验收：** 从公开安装命令完成 Linux/macOS/Windows 安装与 `--version`，记录实际 OS/arch；六平台 CI smoke 可以作补充，不能代替脚本测试。未覆盖的架构明确标未验证。
5. **P0，素材/表单：** 核验至少两张 PH 图、thumbnail 和实际 draft 预览；确认有效个人账户。可先无视频发布；本项目原 45–75 秒视频是建议素材，不是官方门槛。
6. **P1：** 用 5–10 位平台工程师反馈修正文案；补 repo description/homepage、首评、常见问题答复与当天反馈接收渠道。

本次没有运行真实集群扫描、没有向 AI 供应商发送数据，也没有验证未运行平台的公开安装。
