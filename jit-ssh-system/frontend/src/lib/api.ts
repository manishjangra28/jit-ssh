/**
 * Utility to determine the base URL of the Control Plane API.
 * This handles both local development and deployment on a public server.
 */
export const getApiUrl = () => {
  // 1. Check if an environment variable is explicitly set (highest priority)
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  // 2. Client-side dynamic detection
  if (typeof window !== "undefined") {
    const hostname = window.location.hostname;
    
    // If we're on localhost, default to standard development port
    if (hostname === "localhost" || hostname === "127.0.0.1") {
      return "http://localhost:8080/api/v1";
    }

    // If on a public IP or custom domain, assume backend is on port 8080
    // (using the same protocol as the dashboard)
    const protocol = window.location.protocol;
    return `${protocol}//${hostname}:8080/api/v1`;
  }

  // 3. Server-side default (for SSR/Prefetching)
  return "http://localhost:8080/api/v1";
};
