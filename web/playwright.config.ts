import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  reporter: [["list"], ["json", { outputFile: "test-results/results.json" }]],
  timeout: 30000,
  workers: 1,
  use: {
    baseURL: "http://127.0.0.1:18943",
    channel: process.env.PLAYWRIGHT_CHANNEL || "chromium",
    viewport: { width: 1440, height: 1000 },
  },
  webServer: [
    {
      command:
        "../bin/teleskope serve snapshot ../testdata/recorded/eks-demo-snapshot.json --listen 127.0.0.1:18943",
      url: "http://127.0.0.1:18943",
      reuseExistingServer: false,
      timeout: 30000,
    },
    {
      command: "web/.fixtures/demo -serve",
      cwd: "..",
      url: "http://127.0.0.1:8091",
      reuseExistingServer: false,
    },
    {
      command: "web/.fixtures/live-demo -serve 127.0.0.1:18945",
      cwd: "..",
      url: "http://127.0.0.1:18945",
      reuseExistingServer: false,
    },
  ],
});
