import Link from "next/link";

export default function NotFound() {
  return (
    <div className="container card">
      <h1>Page not found</h1>
      <p>The page you are looking for does not exist.</p>
      <Link href="/">Back home</Link>
    </div>
  );
}