import { useState } from "react";
import { appConfig } from "../config";

interface LoginPageProps {
  onLogin: (username?: string, password?: string) => void;
}

export default function LoginPage({ onLogin }: LoginPageProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    const { login } = await import("../store/auth");
    const user = login(username, password);
    if (user) {
      onLogin(user.username, user.password);
    } else {
      setError("用户名或密码错误");
    }
  };

  if (appConfig.authMode === "keycloak") {
    return (
      <div className="login-page">
        <div className="login-form">
          <div className="login-header">
            <h2>K8S AI Ops</h2>
            <p className="eyebrow">AI 运维控制台</p>
          </div>
          <button type="button" className="login-btn" onClick={() => onLogin()}>
            使用 Keycloak 登录
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="login-page">
      <form className="login-form" onSubmit={handleSubmit}>
        <div className="login-header">
          <h2>K8S AI Ops</h2>
          <p className="eyebrow">AI 运维控制台</p>
        </div>

        <div className="form-group">
          <label htmlFor="username">用户名</label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="admin 或 operator"
            autoFocus
          />
        </div>

        <div className="form-group">
          <label htmlFor="password">密码</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="请输入密码"
          />
        </div>

        {error && <div className="login-error">{error}</div>}

        <button type="submit" className="login-btn">
          登 录
        </button>

        <div className="login-hint">
          默认账号：admin / admin123 &nbsp;|&nbsp; operator / operator123
        </div>
      </form>
    </div>
  );
}
