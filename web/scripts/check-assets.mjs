import { execFileSync } from "node:child_process";
execFileSync("git", ["diff", "--exit-code", "--", "internal/report/assets"], {
  cwd: "..",
  stdio: "inherit",
});
const untracked = execFileSync(
  "git",
  ["ls-files", "--others", "--exclude-standard", "internal/report/assets"],
  { cwd: "..", encoding: "utf8" },
).trim();
if (untracked) throw new Error("Untracked generated UI assets: " + untracked);
