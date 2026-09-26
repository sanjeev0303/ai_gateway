"use client";

export default function ErrorPage({
  error,
  reset,
}: {
  readonly error: Error;
  readonly reset: () => void;
}) {
  return (
    <div className="container card">
      <h1>Something went wrong</h1>
      <p>{error.message}</p>
      <button type="button" className="btn" onClick={reset}>
        Try again
      </button>
    </div>
  );
}