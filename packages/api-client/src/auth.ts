import type { RequestOptions, Transport } from "./client";

export interface SessionClaims {
  subject: string;
  email: string;
  display_name: string;
  organization_id: string;
  role: string;
  kind: string;
  issued_at: string;
  expires_at: string;
}

/**
 * Who the caller is. Tokens themselves never appear here: the access and
 * refresh tokens travel only in HttpOnly cookies, and `csrf_token` is present
 * only when CSRF protection is enabled so the browser can echo it in a header.
 */
export interface Session {
  claims: SessionClaims;
  csrf_token?: string;
}

export interface SignUpInput {
  email: string;
  display_name: string;
  organization_name: string;
  password: string;
}

export interface SignInInput {
  email: string;
  password: string;
}

/** Local-development only; enabled by the server's dev-login flag. */
export interface DevLoginInput {
  email: string;
  display_name: string;
}

export interface AuthModule {
  signUp(input: SignUpInput, options?: RequestOptions): Promise<Session>;
  signIn(input: SignInInput, options?: RequestOptions): Promise<Session>;
  devLogin(input: DevLoginInput, options?: RequestOptions): Promise<Session>;
  session(options?: RequestOptions): Promise<Session>;
  refresh(options?: RequestOptions): Promise<Session>;
  logout(options?: RequestOptions): Promise<void>;
}

export function createAuthModule(transport: Transport): AuthModule {
  return {
    signUp: (input, options) =>
      transport.request<Session>("/v1/auth/signup", { ...options, method: "POST", body: input }),
    signIn: (input, options) =>
      transport.request<Session>("/v1/auth/signin", { ...options, method: "POST", body: input }),
    devLogin: (input, options) =>
      transport.request<Session>("/v1/auth/dev-login", { ...options, method: "POST", body: input }),
    session: (options) => transport.request<Session>("/v1/auth/session", options),
    refresh: (options) => transport.request<Session>("/v1/auth/refresh", { ...options, method: "POST" }),
    logout: (options) => transport.request<void>("/v1/auth/logout", { ...options, method: "POST" }),
  };
}