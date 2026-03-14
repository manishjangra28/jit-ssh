"use client";

import { useEffect, useState, useRef } from "react";
import { Bell, BellRing, X, CheckCircle, Info, AlertTriangle, AlertCircle, Clock } from "lucide-react";
import { apiFetch } from "../lib/api";

// Inline helper to avoid any import/module resolution issues reported by user
const getCookieInTray = (name: string): string | null => {
  if (typeof document === "undefined") return null;
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
};

type Notification = {
  id: string;
  title: string;
  message: string;
  type: "info" | "success" | "warning" | "error";
  is_read: boolean;
  created_at: string;
};

type ActivityEvent = {
  id: string;
  username: string;
  type: string;
  remote_ip: string;
  login_time: string;
  server?: { hostname: string };
  user?: { name: string };
};

export default function NotificationTray({ userId }: { userId?: string }) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [activities, setActivities] = useState<ActivityEvent[]>([]);
  const [view, setView] = useState<"notifications" | "activities">("notifications");
  const [isOpen, setIsOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [currentUserId, setCurrentUserId] = useState<string>("");
  const trayRef = useRef<HTMLDivElement>(null);

  // Initialize userId on client side only to avoid hydration mismatch
  useEffect(() => {
    const id = userId || getCookieInTray("jit_auth_id") || "11111111-1111-1111-1111-111111111111";
    console.log("NotificationTray using userId:", id);
    setCurrentUserId(id);
  }, [userId]);

  const fetchData = async () => {
    if (!currentUserId) return;
    try {
      const [notiRes, actRes] = await Promise.all([
        apiFetch(`/notifications?user_id=${currentUserId}`),
        apiFetch("/login-events")
      ]);
      
      if (notiRes.ok) {
        const data = await notiRes.json();
        setNotifications(Array.isArray(data) ? data : []);
        setUnreadCount((Array.isArray(data) ? data : []).filter((n: Notification) => !n.is_read).length);
      }
      if (actRes.ok) {
        const data = await actRes.json();
        setActivities(Array.isArray(data) ? data : []);
      }
    } catch (e) {
      console.error("Failed to fetch notification data", e);
    }
  };

  useEffect(() => {
    if (currentUserId) {
      fetchData();
      const interval = setInterval(fetchData, 10000); // Poll every 10s
      return () => clearInterval(interval);
    }
  }, [currentUserId]);

  const markRead = async (id: string) => {
    try {
      const res = await apiFetch(`/notifications/${id}/read`, { method: "POST" });
      if (res.ok) {
        setNotifications(prev => prev.map(n => n.id === id ? { ...n, is_read: true } : n));
        setUnreadCount(prev => Math.max(0, prev - 1));
      }
    } catch (e) {
      console.error(e);
    }
  };

  const clearAll = async () => {
    if (!currentUserId) return;
    if (!confirm("Are you sure you want to clear all notifications?")) return;
    try {
      const res = await apiFetch(`/notifications?user_id=${currentUserId}`, { method: "DELETE" });
      if (res.ok) {
        setNotifications([]);
        setUnreadCount(0);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const getIcon = (type: string) => {
    switch (type) {
      case "success": return <CheckCircle className="w-4 h-4 text-emerald-500" />;
      case "warning": return <AlertTriangle className="w-4 h-4 text-amber-500" />;
      case "error": return <AlertCircle className="w-4 h-4 text-rose-500" />;
      default: return <Info className="w-4 h-4 text-blue-500" />;
    }
  };

  return (
    <div className="relative" ref={trayRef}>
      <button 
        onClick={() => setIsOpen(!isOpen)}
        className="relative p-2 rounded-full hover:bg-muted/50 transition-colors focus:outline-none group"
      >
        {unreadCount > 0 ? (
          <BellRing className="w-5 h-5 text-primary animate-pulse" />
        ) : (
          <Bell className="w-5 h-5 text-muted-foreground group-hover:text-foreground transition-colors" />
        )}
        {unreadCount > 0 && (
          <span className="absolute top-1 right-1 flex h-4 w-4 items-center justify-center rounded-full bg-rose-500 text-[10px] font-bold text-white shadow-lg border-2 border-background">
            {unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-3 w-80 max-h-[520px] flex flex-col rounded-2xl border border-border/80 bg-[#121212] ring-1 ring-white/10 shadow-[0_20px_50px_rgba(0,0,0,0.5)] z-50 overflow-hidden animate-in fade-in slide-in-from-top-2 duration-200">
          <div className="p-2 border-b border-border bg-muted/30 flex gap-1">
            <button 
              onClick={() => setView("notifications")}
              className={`flex-1 py-1.5 text-xs font-medium rounded-lg transition-colors ${view === 'notifications' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}
            >
              Notifications
            </button>
            <button 
              onClick={() => setView("activities")}
              className={`flex-1 py-1.5 text-xs font-medium rounded-lg transition-colors ${view === 'activities' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}
            >
              Activity
            </button>
          </div>

          <div className="overflow-y-auto divide-y divide-border">
            {view === "notifications" ? (
              notifications.length === 0 ? (
                <div className="p-8 text-center text-sm text-muted-foreground">No notifications yet.</div>
              ) : (
                notifications.map((n) => (
                  <div key={n.id} className={`p-4 hover:bg-muted/20 transition-colors cursor-pointer ${!n.is_read ? 'bg-primary/5' : ''}`} onClick={() => !n.is_read && markRead(n.id)}>
                    <div className="flex gap-3">
                      <div className="mt-1">{getIcon(n.type)}</div>
                      <div className="flex-1">
                        <div className="flex justify-between items-start">
                          <p className={`text-sm font-semibold ${!n.is_read ? 'text-primary' : 'text-foreground'}`}>{n.title}</p>
                          {!n.is_read && <span className="w-2 h-2 rounded-full bg-primary mt-1.5" />}
                        </div>
                        <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{n.message}</p>
                        <p className="text-[10px] text-muted-foreground/60 mt-2">{new Date(n.created_at).toLocaleTimeString()}</p>
                      </div>
                    </div>
                  </div>
                ))
              )
            ) : (
              activities.length === 0 ? (
                <div className="p-8 text-center text-sm text-muted-foreground">No recent activity detected.</div>
              ) : (
                activities.map((a) => (
                  <div key={a.id} className="p-3 hover:bg-muted/10 transition-colors">
                    <div className="flex items-center gap-3">
                      <div className="w-7 h-7 rounded-full bg-emerald-500/10 flex items-center justify-center text-emerald-500 text-[10px] font-bold">
                        {(a.username || "??").slice(0, 2).toUpperCase()}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-[11px] font-medium truncate">
                          {a.user?.name || a.username} <span className="text-muted-foreground">logged into</span> {a.server?.hostname || "Server"}
                        </p>
                        <p className="text-[9px] text-muted-foreground flex items-center gap-1 mt-0.5">
                          <Clock className="w-2.5 h-2.5" /> {new Date(a.login_time).toLocaleString()}
                        </p>
                      </div>
                    </div>
                  </div>
                ))
              )
            )}
          </div>

          <div className="p-3 border-t border-border bg-muted/20 flex justify-between items-center">
             <button onClick={clearAll} className="text-[10px] font-semibold text-rose-500 hover:underline">
               Clear History
             </button>
             <button onClick={() => setIsOpen(false)} className="text-[10px] font-semibold text-primary hover:underline">
               Close Panel
             </button>
          </div>
        </div>
      )}
    </div>
  );
}
