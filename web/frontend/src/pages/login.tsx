import { useState, type FormEvent } from "react";
import { Navigate } from "react-router-dom";
import { Cloud, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth, ApiError } from "@/lib/auth";

export function LoginPage() {
  const { user, needsSetup, isLoading, login, setup } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Already authenticated — redirect to files
  if (user) {
    return <Navigate to="/files" replace />;
  }

  if (needsSetup) {
    return <SetupForm onSetup={setup} />;
  }

  return <LoginForm onLogin={login} />;
}

function LoginForm({
  onLogin,
}: {
  onLogin: (username: string, password: string) => Promise<void>;
}) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");

    if (!username.trim() || !password) {
      setError("Username and password are required.");
      return;
    }

    setSubmitting(true);
    try {
      await onLogin(username.trim(), password);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <Cloud className="mx-auto h-10 w-10 text-primary" />
          <h1 className="mt-4 text-2xl font-semibold tracking-tight text-foreground">
            Sign in to MiniCloud
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Enter your credentials to continue
          </p>
        </div>

        <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
          <form className="space-y-4" onSubmit={handleSubmit}>
            {error && (
              <div className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">
                {error}
              </div>
            )}
            <Input
              label="Username"
              placeholder="admin"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={submitting}
            />
            <Input
              label="Password"
              type="password"
              placeholder="••••••••"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={submitting}
            />
            <Button className="w-full" type="submit" disabled={submitting}>
              {submitting && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Sign In
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}

function SetupForm({
  onSetup,
}: {
  onSetup: (
    token: string,
    username: string,
    password: string,
  ) => Promise<void>;
}) {
  const [token, setToken] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");

    if (!token.trim()) {
      setError("Setup token is required.");
      return;
    }
    if (!username.trim()) {
      setError("Username is required.");
      return;
    }
    if (!password || password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }

    setSubmitting(true);
    try {
      await onSetup(token.trim(), username.trim(), password);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <Cloud className="mx-auto h-10 w-10 text-primary" />
          <h1 className="mt-4 text-2xl font-semibold tracking-tight text-foreground">
            Welcome to MiniCloud
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Create your admin account to get started
          </p>
        </div>

        <div className="rounded-xl border border-border bg-surface p-6 shadow-sm">
          <form className="space-y-4" onSubmit={handleSubmit}>
            {error && (
              <div className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">
                {error}
              </div>
            )}
            <Input
              label="Setup Token"
              placeholder="Paste the token from your server logs"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              disabled={submitting}
            />
            <Input
              label="Username"
              placeholder="admin"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={submitting}
            />
            <Input
              label="Password"
              type="password"
              placeholder="••••••••"
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={submitting}
            />
            <Button className="w-full" type="submit" disabled={submitting}>
              {submitting && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              Create Account
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
