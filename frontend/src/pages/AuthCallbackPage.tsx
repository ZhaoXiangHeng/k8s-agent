import { useEffect, useState } from "react";
import { Notice } from "../components/Notice";

export function AuthCallbackPage({ onCallback }: { onCallback: (search: string) => Promise<void> }) {
  const [error, setError] = useState("");

  useEffect(() => {
    onCallback(window.location.search)
      .then(() => window.history.replaceState({}, "", "/"))
      .catch((err) => setError(err instanceof Error ? err.message : "登录回调失败"));
  }, [onCallback]);

  return (
    <main className="loginPage">
      <section className="loginPanel">
        <h1>正在完成登录</h1>
        {error ? <Notice type="error">{error}</Notice> : <p className="subtle">请稍候...</p>}
      </section>
    </main>
  );
}
