import { describe, expect, test } from "bun:test";
import { parseReviewsEnabled } from "./feature-flags";

describe("parseReviewsEnabled", () => {
  test("defaults on outside production", () => {
    expect(parseReviewsEnabled(undefined, "development")).toBe(true);
    expect(parseReviewsEnabled(undefined, "test")).toBe(true);
    expect(parseReviewsEnabled(undefined, undefined)).toBe(true);
  });

  test("defaults off in production", () => {
    expect(parseReviewsEnabled(undefined, "production")).toBe(false);
    expect(parseReviewsEnabled("", "production")).toBe(false);
  });

  test("explicit enable wins over the production default", () => {
    expect(parseReviewsEnabled("1", "production")).toBe(true);
    expect(parseReviewsEnabled("true", "production")).toBe(true);
    expect(parseReviewsEnabled(" TRUE ", "production")).toBe(true);
  });

  test("explicit disable wins over the development default", () => {
    expect(parseReviewsEnabled("0", "development")).toBe(false);
    expect(parseReviewsEnabled("false", "development")).toBe(false);
  });

  test("unrecognized values fall back to the environment default", () => {
    expect(parseReviewsEnabled("yes", "production")).toBe(false);
    expect(parseReviewsEnabled("yes", "development")).toBe(true);
  });
});
