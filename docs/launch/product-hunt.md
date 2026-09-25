# Product Hunt Launch Workspace

This document keeps Teleskope's Product Hunt launch work in one place while the
product continues to mature. Update it as features, releases, screenshots, and
positioning become ready.

## Launch Decision

Latest audit: [2026-09-25 readiness review](ph-readiness-2026-09-25.md).
Current decision: **NO-GO for scheduling**. Public v0.15.0 downloads and six-platform
binary smoke checks are verified; a license decision, publicly hosted interactive
demo, release/demo alignment, consolidated data-handling note, published-install
verification, and Product Hunt account/gallery confirmation remain open. Three README
[highlight GIFs](../../assets/highlights/) now cover topology, evidence, and fleet
review. They are nine-second UI keyframe walkthroughs; a dedicated Product Hunt
gallery and upload preview still need verification.

Teleskope is a good fit for Product Hunt once it is easy for a new visitor to
try or inspect the product without preparing a real Kubernetes cluster.

The launch should optimize for the first serious users: platform engineers,
SREs, Kubernetes operators, and consultants who need to understand a cluster
before an upgrade, migration, incident review, or cleanup.

Do not treat leaderboard rank as the main success metric. The better outcome is
specific feedback from people who maintain clusters and can tell us which
inventory, comparison, and reporting gaps block adoption.

## Positioning

Primary message:

> Understand your Kubernetes clusters before you change them.

One-line description:

> Teleskope turns EKS and Kubernetes inventory into local reports, topology
> views, and migration comparisons that teams can review before touching a
> cluster.

Short tagline options:

- Kubernetes inventory before migrations, upgrades, and cleanups.
- A local review room for Kubernetes cluster changes.
- See the shape of a cluster before you change it.

Category:

- Developer Tools
- Open Source, only after the owner selects a license and adds `LICENSE`
- Artificial Intelligence, only if the optional LLM analysis becomes central to
  the launch story

Audience:

- Platform engineers planning EKS or Kubernetes migrations
- SREs reviewing production cluster risk before changes
- Consultants and infra teams who need shareable evidence without broad cluster
  access

## Proof Points

- Collects EKS metadata and Kubernetes API inventory into local report
  artifacts.
- Produces a static interactive report with filters, topology, and resource
  details.
- Records coverage and collection errors so partial inventory is visible.
- Avoids collecting Kubernetes Secret values.
- Compares source and target scan reports for migration planning.
- Keeps LLM analysis optional and separate from deterministic scan, compare,
  serve, and export workflows.
- Offers local live inventory and a multi-cluster hub for fleet review.

## Product Hunt Page Draft

Name:

> Teleskope

Tagline:

> Understand your Kubernetes clusters before you change them.

Description:

> Inspect EKS and Kubernetes inventory, topology, and migration gaps in local
> reports. Share evidence without granting cluster access. Try a synthetic demo;
> optional AI analysis stays separate from deterministic findings.

Maker comment draft:

> Hi Product Hunt. We built Teleskope for the awkward moment before a Kubernetes
> migration, upgrade, incident review, or cleanup: everyone needs to understand
> the cluster, but the knowledge is scattered across APIs, cloud metadata,
> manifests, and tribal memory.
>
> Teleskope scans EKS and Kubernetes, writes local report artifacts, and gives
> teams an interactive view of workloads, topology, platform capabilities, and
> collection coverage. It can also compare two scan reports so migration gaps
> are visible before the change window.
>
> The product is early, and we are especially looking for feedback from platform
> engineers, SREs, and Kubernetes consultants. The most useful feedback for us:
> which cluster facts you need before a migration, which report views would save
> time in reviews, and what would make this safe enough to run in your own
> environment.

## Assets To Prepare

Project launch assets to prepare (the video is optional; see the latest audit
for actual Product Hunt requirements):

- Public sample report built from realistic demo data.
- Download link for a stable release.
- Landing page with a clear demo, install command, and GitHub link.
- Product Hunt gallery images: three to five screenshots or short visual cards.
- Optional short demo video, ideally 45 to 75 seconds; it is not a PH requirement.
- Clear privacy and security note covering local reports, credentials, Secrets,
  ConfigMaps, live server exposure, and optional LLM analysis.
- GitHub README section that points visitors to the sample report and release.

## Demo Storyboard

The launch video should show the product, not a conceptual slide deck.

1. Start with a cluster change question: "What will break if we migrate this
   cluster?"
2. Run a scan command and show report files being produced.
3. Open the static report and show topology, filters, and resource details.
4. Show advisor evidence and collection coverage.
5. Compare source and target scan reports.
6. End on the shareable local report and GitHub/install call to action.

## Ongoing Readiness Checklist

Launch later, but keep these current while features are still moving:

- [ ] Decide the first launch version number (published latest: v0.15.0).
- [ ] Select a license and add `LICENSE` before using the Open Source positioning.
- [x] Verify a public stable release and six-platform native binary smoke checks
  (v0.15.0; this does not verify the public install scripts).
- [ ] Verify install scripts against the published release on macOS, Linux, and
  Windows.
- [x] Generate a realistic synthetic demo dataset ([samples and reproduction](demo-assets.md)).
- [ ] Publish a public interactive sample report (committed HTML exists; Pages
  was not configured at the latest audit).
- [x] Capture current synthetic UI highlights ([GIFs and nine still frames](../../assets/highlights/README.md)).
- [ ] Export dedicated PH gallery assets and verify the submission preview.
- [ ] Optionally record the demo video against the latest release candidate.
- [ ] Add a concise security and data-handling note.
- [ ] Prepare a landing page or GitHub Pages site.
- [ ] Prepare Product Hunt copy from the current feature set.
- [ ] Prepare launch-day reply snippets for common questions.
- [ ] Ask five to ten Kubernetes practitioners for private feedback before
  launch day.
- [ ] Confirm Product Hunt account access and maker profiles.

## Inputs Needed From Us

These inputs require product or business judgment:

- Launch timing: target month or target feature milestone.
- Primary CTA: GitHub star, download release, open sample report, or join a
  feedback list.
- Demo data policy: synthetic only, sanitized real cluster, or both.
- Product maturity line: what should be called stable, preview, or experimental.
- AI positioning: whether optional LLM analysis is a side feature or part of the
  main story.
- Contact path: GitHub issues, Discord/Slack, email, or a form.

Everything else can be prepared directly from the repo: draft copy, screenshot
list, demo script, release checklist, README updates, landing page structure,
and Product Hunt page text.

## Launch Risks

- A visitor without Kubernetes access needs a sample report, or the product will
  feel hard to evaluate.
- Security-sensitive buyers will want clear language about what data is
  collected and where it goes.
- The audience is technical and narrower than many Product Hunt launches, so
  specific community feedback matters more than broad upvotes.
- If live hub features are still early, keep them in the proof points but avoid
  making them the only reason to try the product.

