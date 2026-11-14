// lab.sysinfo — first plugin: registers one tool + proves load/reload lifecycle.
// Plain ESM JavaScript, no dependencies: this machine has no node/npm, and
// OpenCode runs plugin files in its own runtime.

import fs from "node:fs";
import os from "node:os";

const LOG = "/tmp/opencode/lab-sysinfo.log";

function log(message) {
  const line = `${new Date().toISOString()} ${message}\n`;
  try {
    fs.appendFileSync(LOG, line);
  } catch {}
  console.log(`[lab.sysinfo] ${message}`);
}

function gib(bytes) {
  return `${(bytes / 1024 ** 3).toFixed(2)} GiB`;
}

export default {
  id: "lab.sysinfo",

  async setup(ctx) {
    log(`loaded at ${ctx.location?.directory ?? "?"} (OpenCode ${ctx.app?.version ?? "?"})`);

    await ctx.tool.transform((editor) => {
      editor.add({
        name: "sysinfo",
        description:
          "Report disk, memory, platform, and working directory of the machine running OpenCode",
        input: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "Directory to report on (defaults to the process working directory)",
            },
          },
          additionalProperties: false,
        },
        options: { namespace: "lab", codemode: true },
        execute: async (input) => {
          const dir = input?.path || process.cwd();
          let disk;
          try {
            const s = fs.statfsSync(dir);
            disk = `${gib(s.bavail * s.bsize)} free of ${gib(s.blocks * s.bsize)}`;
          } catch (err) {
            disk = `unavailable (${err.message})`;
          }
          return {
            content: [
              `directory: ${dir}`,
              `disk: ${disk}`,
              `memory: ${gib(os.freemem())} free of ${gib(os.totalmem())}`,
              `platform: ${os.platform()} ${os.arch()} (${os.release()})`,
              `uptime: ${(os.uptime() / 3600).toFixed(1)} h`,
            ].join("\n"),
          };
        },
      });
    });

    return () => log("unloaded");
  },
};
