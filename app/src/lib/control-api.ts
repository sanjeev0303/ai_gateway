export const CONTROL_API_BASE =
  process.env.NEXT_PUBLIC_CONTROL_API_BASE ??
  (typeof window !== "undefined"
    ? `${window.location.origin}/api/control`
    : (process.env.NEXT_PUBLIC_CONTROL_API_URL || "http://localhost:8081"));