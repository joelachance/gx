import type { NextConfig } from "next";
import { createMDX } from "fumadocs-mdx/next";

const nextConfig: NextConfig = {
  // A static export, not a server. There is no database, no auth provider and
  // no API route left in this app — the one page that used to redirect signed-in
  // visitors became sync when its session check went, which is what removed the
  // last thing forcing a Node runtime. That is what lets gx.run be served by
  // Cloudflare Pages off a build artifact, with nothing to run and nothing to
  // pay for.
  output: "export",
  images: { unoptimized: true },
};

const withMDX = createMDX();

export default withMDX(nextConfig);
