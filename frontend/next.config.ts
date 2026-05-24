import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone", // required for Docker production build
};

export default nextConfig;
