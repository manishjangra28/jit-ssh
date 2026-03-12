import { useState } from "react";
import { ShieldCheck, Eye, EyeOff, ArrowRight } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { getApiUrl } from "@/lib/api";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPwd, setShowPwd] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${getApiUrl()}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      const data = await res.json();

      if (!res.ok) {
        if (data.error === "password_not_set") {
          setError("Your account has no password yet. Please ask the admin to set one for you.");
        } else {
          setError(data.error || "Login failed");
        }
        return;
      }

      // Store auth info in cookies (max-age = 24 hours)
      document.cookie = `jit_auth_role=${data.role}; path=/; max-age=86400`;
      document.cookie = `jit_auth_name=${encodeURIComponent(data.name || data.email)}; path=/; max-age=86400`;
      document.cookie = `jit_auth_id=${data.id}; path=/; max-age=86400`;
      document.cookie = `jit_auth_email=${encodeURIComponent(data.email)}; path=/; max-age=86400`;

      // Redirect: admin/approver → /admin, developer → /
      const isAdminPortal = data.role === "admin" || data.role === "approver";
      window.location.href = isAdminPortal ? "/admin" : "/";
    } catch (err) {
      setError("Network error — is the server running?");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-4">
      <div className="mb-8 flex flex-col items-center">
        <div className="w-16 h-16 bg-gradient-to-tr from-blue-600 to-indigo-500 rounded-2xl flex items-center justify-center shadow-lg mb-4">
          <ShieldCheck className="w-8 h-8 text-white" />
        </div>
        <h1 className="text-3xl font-bold tracking-tight">JIT SSH System</h1>
        <p className="text-muted-foreground mt-2">Sign in with your email and password</p>
      </div>

      <Card className="w-full max-w-md">
        <CardHeader className="pb-2">
          <CardTitle>Welcome back</CardTitle>
          <CardDescription>Works for all roles — Developer, Approver, Admin</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleLogin} className="space-y-4">
            <div className="space-y-1">
              <label className="text-sm font-medium">Email address</label>
              <input
                type="email"
                required
                autoFocus
                className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="you@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>

            <div className="space-y-1">
              <label className="text-sm font-medium">Password</label>
              <div className="relative">
                <input
                  type={showPwd ? "text" : "password"}
                  required
                  className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="••••••••"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
                <button
                  type="button"
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  onClick={() => setShowPwd(!showPwd)}
                >
                  {showPwd ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

            {error && (
              <div className="text-sm text-red-400 bg-red-400/10 border border-red-400/20 rounded-md px-3 py-2">
                {error}
              </div>
            )}

            <Button type="submit" className="w-full gap-2" disabled={loading}>
              {loading ? "Signing in..." : (<>Sign In <ArrowRight className="w-4 h-4" /></>)}
            </Button>
          </form>

          <div className="mt-4 pt-4 border-t border-border text-xs text-muted-foreground space-y-1">
            <p>🔐 <strong>Admins</strong> and <strong>Approvers</strong> will be redirected to <code>/admin</code></p>
            <p>💻 <strong>Developers</strong> will be redirected to the user portal</p>
            <p className="pt-1">No password? Ask your admin to set one via the User Management panel.</p>
          </div>
        </CardContent>
      </Card>

      <p className="mt-8 text-sm text-muted-foreground">Internal Network Access Only</p>
    </div>
  );
}
