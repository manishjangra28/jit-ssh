"use client";

import { Inter } from "next/font/google";
import "./globals.css";
import {
  Home,
  Server,
  Settings,
  Bell,
  Search,
  UserCircle,
  LogOut,
  TerminalSquare,
  Users,
  Layers,
  Clock,
  ShieldAlert,
  Cloud,
} from "lucide-react";
import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";

const inter = Inter({ subsets: ["latin"] });

// Mock utility to read cookies on the client side
function getCookie(name: string) {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp("(^| )" + name + "=([^;]+)"));
  if (match) return match[2];
  return null;
}

import NotificationTray from "../components/NotificationTray";

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const router = useRouter();
  const pathname = usePathname();

  const [auth, setAuth] = useState<{
    role: string | null;
    name: string | null;
    id: string | null;
    loaded: boolean;
  }>({
    role: null,
    name: null,
    id: null,
    loaded: false,
  });

  useEffect(() => {
    const checkSession = () => {
      const role = getCookie("jit_auth_role");
      const name = getCookie("jit_auth_name");
      const id = getCookie("jit_auth_id");

      if (!role && pathname !== "/login") {
        router.push("/login");
        return;
      }

      setAuth((prev) => {
        if (
          prev.role === role &&
          prev.name === name &&
          prev.id === id &&
          prev.loaded
        ) {
          return prev;
        }
        return { role, name, id, loaded: true };
      });

      if (role === "developer" && pathname.startsWith("/admin")) {
        router.push("/");
      } else if ((role === "admin" || role === "approver") && pathname === "/") {
        router.push("/admin");
      }
    };

    checkSession();
    // Automatically redirect to login if session cookies expire (set to 10 mins on login)
    const interval = setInterval(checkSession, 10000);
    return () => clearInterval(interval);
  }, [pathname, router]);

  const handleLogout = () => {
    document.cookie = "jit_auth_role=; path=/; max-age=0";
    document.cookie = "jit_auth_name=; path=/; max-age=0";
    document.cookie = "jit_auth_id=; path=/; max-age=0";
    document.cookie = "jit_auth_email=; path=/; max-age=0";
    document.cookie = "jit_auth_token=; path=/; max-age=0";
    router.push("/login");
  };

  if (!auth.loaded) {
    return (
      <html lang="en" className="dark">
        <body
          className={`${inter.className} min-h-screen bg-background antialiased flex items-center justify-center`}
        >
          <div className="w-6 h-6 border-4 border-primary border-t-transparent rounded-full animate-spin"></div>
        </body>
      </html>
    );
  }

  // Login page layout without sidebar
  if (pathname === "/login") {
    return (
      <html lang="en" className="dark">
        <body
          className={`${inter.className} min-h-screen bg-background antialiased`}
        >
          {children}
        </body>
      </html>
    );
  }

  const isAdminPortal = auth.role === "admin" || auth.role === "approver";

  return (
    <html lang="en" className="dark">
      <body
        className={`${inter.className} min-h-screen bg-background antialiased flex`}
      >
        {/* Sidebar */}
        <aside className="w-64 border-r bg-card/50 backdrop-blur-xl hidden md:flex flex-col">
          <div className="p-6">
            <h1 className="text-xl font-bold bg-gradient-to-r from-blue-500 to-indigo-500 bg-clip-text text-transparent flex items-center gap-2">
              <ShieldAlert className="w-6 h-6 text-blue-500" />
              {isAdminPortal ? "JIT Admin" : "JIT Portal"}
            </h1>
          </div>

          <nav className="flex-1 px-4 space-y-1 mt-4">
            {isAdminPortal ? (
              <>
                <a
                  href="/admin"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/admin" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Home className="w-4 h-4" /> Dashboard
                </a>
                <a
                  href="/admin/servers"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/admin/servers" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Server className="w-4 h-4" /> Servers & Clusters
                </a>
                <a
                  href="/admin/cloud"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname.startsWith("/admin/cloud") ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Cloud className="w-4 h-4" /> Cloud Access
                </a>
                <a
                  href="/admin/users"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/admin/users" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Users className="w-4 h-4" /> User Management
                </a>
                <a
                  href="/admin/tokens"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/admin/tokens" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <ShieldAlert className="w-4 h-4" /> Agent Tokens
                </a>
                <a
                  href="/admin/teams"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/admin/teams" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Layers className="w-4 h-4" /> Teams & Groups
                </a>
                <a
                  href="/admin/audit"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/admin/audit" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Clock className="w-4 h-4" /> Audit Logs
                </a>
                <a
                  href="/admin/settings"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/admin/settings" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Settings className="w-4 h-4" /> Settings
                </a>
              </>
            ) : (
              <>
                <a
                  href="/"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <TerminalSquare className="w-4 h-4" /> My Sessions & Requests
                </a>
                <a
                  href="/cloud"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname.startsWith("/cloud") ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Cloud className="w-4 h-4" /> Cloud Access
                </a>
                <a
                  href="/settings"
                  className={`flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors ${pathname === "/settings" ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground"}`}
                >
                  <Settings className="w-4 h-4" /> Settings
                </a>
              </>
            )}
          </nav>

          <div className="p-4 border-t">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <UserCircle className="w-8 h-8 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium truncate w-32">
                    {auth.name}
                  </p>
                  <p className="text-[10px] text-muted-foreground uppercase">
                    {auth.role}
                  </p>
                </div>
              </div>
              <button
                onClick={handleLogout}
                className="p-2 text-muted-foreground hover:text-destructive transition-colors"
                title="Logout"
              >
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <div className="flex-1 flex flex-col min-w-0">
          <header className="h-16 border-b bg-card/50 backdrop-blur-xl flex items-center justify-between px-6 sticky top-0 z-10 w-full">
            <div className="flex items-center bg-muted/50 rounded-md px-3 py-1.5 w-64 border border-border">
              <Search className="w-4 h-4 text-muted-foreground mr-2" />
              <input
                type="text"
                placeholder={
                  isAdminPortal ? "Search servers or users..." : "Search servers..."
                }
                className="bg-transparent border-none outline-none text-sm placeholder:text-muted-foreground w-full"
              />
            </div>

            <div className="flex items-center gap-4">
              <NotificationTray userId={auth.id || undefined} />
            </div>
          </header>

          <main className="flex-1 overflow-auto p-6 md:p-8">
            <div className="max-w-6xl mx-auto space-y-8">{children}</div>
          </main>
        </div>
      </body>
    </html>
  );
}
