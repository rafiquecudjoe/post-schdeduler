import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  async rewrites() {
    return [];
  },
  experimental: {
    proxyTimeout: 300000, // 5 minutes for SSE connections
  },
};

export default nextConfig;
