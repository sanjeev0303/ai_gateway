import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Sign in",
};

export default function LoginPage() {
  return (
    <>
      <h1>Sign in</h1>
      <form className="form">
        <label>
          Email
          <input type="email" name="email" autoComplete="email" required />
        </label>
        <label>
          Password
          <input type="password" name="password" autoComplete="current-password" required />
        </label>
        <button type="submit" className="btn">
          Sign in
        </button>
      </form>
    </>
  );
}