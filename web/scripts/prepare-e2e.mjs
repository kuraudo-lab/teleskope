import { mkdirSync } from "node:fs";
import { execFileSync } from "node:child_process";
mkdirSync(".fixtures", { recursive: true });
const go = (args) =>
  execFileSync("go", args, { cwd: "..", stdio: "inherit", env: process.env });
go(["build", "-o", "web/.fixtures/demo", "./scripts/demo"]);
go(["build", "-o", "web/.fixtures/live-demo", "./scripts/topology-live-demo"]);
for (const profile of ["small", "medium", "large"])
  go([
    "run",
    "./scripts/topology-scale",
    "-profile",
    profile,
    "-output",
    `web/.fixtures/${profile}.html`,
  ]);
