import { readFileSync } from "node:fs";
import path from "node:path";

const DOCS_DIR = path.join(process.cwd(), "docs");

export function readDocsFile(name: "llms.txt" | "llms-full.txt"): string {
  return readFileSync(path.join(DOCS_DIR, name), "utf8");
}
