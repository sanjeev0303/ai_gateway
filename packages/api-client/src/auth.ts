import type { components } from "@ai-governance/contracts";
import type { RequestOptions, Transport } from "./client";

/**
 * Who the caller is. Tokens never appear here: the access and refresh tokens
 * travel only in HttpOnly cookies.
 */
export type Session = components["schemas"]["AuthSession"];
export type SignUpInput = components["schemas"]["SignUpRequest"];
export type SignInInput = components["schemas"]["SignInRequest"];
/** Local-development only; enabled by the server's dev-login flag. */
export type DevLoginInput = components["schemas"]["DevLoginRequest"];

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