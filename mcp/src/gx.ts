import { execFile } from "node:child_process";
import { existsSync, statSync } from "node:fs";
import { homedir } from "node:os";
import { join, resolve } from "node:path";

export type GxRunOptions = {
  cwd?: string;
  sessionId?: string;
  sessionIds?: string[];
  timeoutMs?: number;
};

export type GxRunResult = {
  command: string[];
  cwd: string;
  exitCode: number;
  stdout: string;
  stderr: string;
};

export type FormatOptions = {
  action?: string;
  display?: string;
  result?: unknown;
  stderr?: string;
  nextActions?: string[];
  extra?: Record<string, unknown>;
};

export function resolveTlBinary() {
  const override = (process.env.GX_BINARY || "").trim();
  if (override) {
    return override;
  }
  const installed = join(homedir(), ".local", "bin", "gx");
  if (existsSync(installed)) {
    return installed;
  }
  return "gx";
}

function resolveCwd(raw?: string) {
  const cwd = resolve(raw || process.cwd());
  let stat;
  try {
    stat = statSync(cwd);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`cwd does not exist: ${cwd}: ${message}`);
  }
  if (!stat.isDirectory()) {
    throw new Error(`cwd is not a directory: ${cwd}`);
  }
  return cwd;
}

export function commandEnvironment() {
  const env = { ...process.env };
  env.GX_REVIEW_AI = "1";
  env.GX_MCP = "1";
  const pathEntries = [
    join(homedir(), ".local", "bin"),
    env.PATH || "",
    "/opt/homebrew/bin",
    "/usr/local/bin",
    "/opt/local/bin",
    "/usr/bin",
    "/bin",
  ].flatMap((entry) => entry.split(":").filter(Boolean));
  env.PATH = [...new Set(pathEntries)].join(":");
  return env;
}

export function runTt(args: string[], options: GxRunOptions = {}): Promise<GxRunResult> {
  const cwd = resolveCwd(options.cwd);
  const env = commandEnvironment();
  const binary = resolveTlBinary();
  const sessionIds = [...(options.sessionIds || [])];
  if (options.sessionId) {
    sessionIds.unshift(options.sessionId);
  }
  const cleanedSessionIds = [...new Set(sessionIds.map((id) => id.trim()).filter(Boolean))];
  if (cleanedSessionIds.length === 1) {
    env.GX_SESSION_ID = cleanedSessionIds[0];
  } else if (cleanedSessionIds.length > 1) {
    env.GX_SESSION_IDS = cleanedSessionIds.join(",");
  }

  return new Promise((resolve, reject) => {
    execFile(
      binary,
      args,
      {
        cwd,
        env,
        timeout: options.timeoutMs ?? 120_000,
        maxBuffer: 10 * 1024 * 1024,
      },
      (error, stdout, stderr) => {
        const exitCode =
          typeof (error as NodeJS.ErrnoException | null)?.code === "number"
            ? Number((error as NodeJS.ErrnoException).code)
            : error
              ? 1
              : 0;
        const result = {
          command: [binary, ...args],
          cwd,
          exitCode,
          stdout: redact(stdout.trim(), env),
          stderr: redact(stderr.trim(), env),
        };
        if (error) {
          reject(Object.assign(new Error(result.stderr || result.stdout || error.message), { result }));
          return;
        }
        resolve(result);
      },
    );
  });
}

function redact(value: string, env: NodeJS.ProcessEnv) {
  let out = value;
  const names = [
    "OPENAI_API_KEY",
    "GX_OPENAI_API_KEY",
    "ANTHROPIC_API_KEY",
    "GX_UPLOAD_TOKEN",
    "AWS_SECRET_ACCESS_KEY",
  ];
  for (const name of names) {
    const secret = env[name]?.trim();
    if (secret && secret.length >= 8) {
      out = out.split(secret).join(`[REDACTED:${name}]`);
    }
  }
  return out;
}

function envelope(result: GxRunResult, options: FormatOptions = {}) {
  return {
    ok: result.exitCode === 0,
    action: options.action,
    command: result.command,
    cwd: result.cwd,
    exit_code: result.exitCode,
    display: options.display ?? result.stdout,
    result: options.result ?? (result.stdout || null),
    stderr: options.stderr ?? (result.stderr || undefined),
    next_actions: options.nextActions,
    ...(options.extra ?? {}),
  };
}

export function formatResult(result: GxRunResult, options: FormatOptions = {}): string {
  return JSON.stringify(envelope(result, options), null, 2);
}

export function formatError(error: unknown, options: FormatOptions = {}): string {
  const maybeResult = (error as { result?: GxRunResult }).result;
  if (maybeResult) {
    return formatResult(maybeResult, withKeyGuidance(`${maybeResult.stderr}\n${maybeResult.stdout}`, options));
  }
  const message = error instanceof Error ? error.message : String(error);
  const guided = withKeyGuidance(message, options);
  return JSON.stringify(
    {
      ok: false,
      action: guided.action,
      command: undefined,
      cwd: guided.extra?.cwd ?? undefined,
      exit_code: 1,
      display: guided.display ?? message,
      result: null,
      stderr: guided.stderr ?? message,
      next_actions: guided.nextActions,
      ...(guided.extra ?? {}),
    },
    null,
    2,
  );
}

function withKeyGuidance(message: string, options: FormatOptions): FormatOptions {
  const guidance = keyGuidance(message);
  if (!guidance) {
    return options;
  }
  return {
    ...options,
    display: options.display ?? guidance.display,
    nextActions: prependNextAction(options.nextActions, guidance.nextAction),
    extra: {
      ...(options.extra ?? {}),
      key_required: true,
    },
  };
}

// keyGuidance turns gx's "no model key" refusal into the fix that works for an
// MCP server. The server is spawned by its host application and inherits that
// application's environment, not the user's shell, so a key exported in a
// terminal is invisible here; it has to be set in the server's own config.
function keyGuidance(message: string): { display: string; nextAction: string } | undefined {
  const text = message.toLowerCase();
  if (!text.includes("anthropic_api_key") && !text.includes("openai_api_key")) {
    return undefined;
  }
  const nextAction =
    "Set ANTHROPIC_API_KEY or OPENAI_API_KEY in the env block of the gx MCP server's config (it does not inherit your shell's environment), then retry the MCP tool.";
  return {
    display: `gx review needs a model API key. ${nextAction}`,
    nextAction,
  };
}

function prependNextAction(existing: string[] | undefined, action: string): string[] {
  const actions = [action, ...(existing ?? [])];
  return actions.filter((item, index) => actions.indexOf(item) === index);
}
