# Issue #31: public synthetic sample reports

Request: prepare public reports for EKS, Gateway, Storage, Hub, and AI analysis,
with no real sensitive information and documented README entry points.

Use newly authored synthetic fixtures, never a real scan as the generation input.
Generate deterministic source/target HTML reports through the production renderer,
a fleet summary through the production hub model, and an explicitly hand-authored
AI output example through the production analysis model/Markdown renderer. Keep
AI narrative distinct from collected facts; no model call or credential is needed.
Provide committed browser-readable artifacts, raw JSON, a catalog, reproducible
Go generation, and an optional loopback-only real Hub preview seeded with fixtures.
Document intentional differences and partial coverage. Do not claim a hosted URL
or actual AI execution. Verify relationships, artifact reproducibility, privacy
constraints, browser navigation, make check/build, and diff whitespace.
