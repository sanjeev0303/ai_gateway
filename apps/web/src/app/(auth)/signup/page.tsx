import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Create account",
};

export default function SignupPage() {
  return (
    <>
      <h1>Create your workspace</h1>
      <form className="form">
        <label>
          Organization name
          <input type="text" name="organization" autoComplete="organization" required />
        </label>
        <label>
          Email
          <input type="email" name="email" autoComplete="email" required />
        </label>
        <label>
          Password
          <input type="password" name="password" autoComplete="new-password" required />
        </label>
        <button type="submit" className="btn">
          Create workspace
        </button>
      </form>
    </>
  );
}