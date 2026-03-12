"use client";

import { useEffect, useState } from "react";
import { Server, Activity, Users, ShieldAlert, CheckCircle2, XCircle, RefreshCw, Clock, Key } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getApiUrl } from "@/lib/api";

export default function AdminDashboardPage() {
  const [servers, setServers] = useState<any[]>([]);
  const [requests, setRequests] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [srvRes, reqRes] = await Promise.all([
        fetch(`${getApiUrl()}/servers`),
        fetch(`${getApiUrl()}/requests`)
      ]);
      if (srvRes.ok) setServers(await srvRes.json());
      if (reqRes.ok) setRequests(await reqRes.json());
    } catch (e) {
      console.error("Failed to fetch data", e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const intv = setInterval(fetchData, 10000);
    return () => clearInterval(intv);
  }, []);

  const handleApprove = async (id: string, duration?: string) => {
    try {
      const authId = document.cookie.split("; ").find(r => r.startsWith("jit_auth_id="))?.split("=")[1];
      const res = await fetch(`${getApiUrl()}/requests/${id}/approve`, {
        method: 'POST',
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ 
          approver_id: authId,
          duration: duration 
        })
      });
      if (!res.ok) {
        const err = await res.json();
        alert(err.error || "Failed to approve request");
      } else {
        fetchData();
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleRevoke = async (id: string) => {
    if (!confirm("Are you sure you want to revoke this access immediately?")) return;
    try {
      const res = await fetch(`${getApiUrl()}/requests/${id}/revoke`, {
        method: 'POST',
        headers: { "Content-Type": "application/json" }
      });
      if (res.ok) {
        fetchData();
      } else {
        const err = await res.json();
        alert(err.error || "Failed to revoke access");
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleReject = async (id: string, currentlyPending: boolean) => {
    if (!currentlyPending) return;
    // Mock rejection / mock deletion for demo purposes if rejection API doesn't exist
    alert("Rejection flow needs backend endpoint: DELETE /requests/" + id);
  };

  const pendingRequests = requests.filter(r => r.status === "pending");
  const activeSessions = requests.filter(r => r.status === "approved").length;

  return (
    <>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Admin Overview</h2>
          <p className="text-muted-foreground mt-1">Management and observability for JIT infrastructure access.</p>
        </div>
        <Button onClick={fetchData} variant="outline" size="icon" disabled={loading}>
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {/* Metrics Row */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="bg-card/60 backdrop-blur-sm border-border hover:border-primary/50 transition-colors">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Servers</CardTitle>
            <Server className="w-4 h-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-foreground">{servers.length}</div>
            <p className="text-xs text-muted-foreground mt-1">
              <span className="text-emerald-500 font-medium">{servers.filter(s => s.status === 'online').length}</span> online
            </p>
          </CardContent>
        </Card>

        <Card className="bg-card/60 backdrop-blur-sm border-border hover:border-primary/50 transition-colors">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-sm font-medium text-muted-foreground">Active Sessions</CardTitle>
            <Activity className="w-4 h-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-foreground">{activeSessions}</div>
            <p className="text-xs text-muted-foreground mt-1">+12% from last week</p>
          </CardContent>
        </Card>

        <Card className="bg-card/60 backdrop-blur-sm border-border hover:border-primary/50 transition-colors relative overflow-hidden">
          <div className="absolute top-0 right-0 w-32 h-32 bg-indigo-500/10 rounded-full blur-3xl -mr-10 -mt-10"></div>
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0 relative z-10">
            <CardTitle className="text-sm font-medium text-muted-foreground">Pending Approvals</CardTitle>
            <ShieldAlert className={`w-4 h-4 ${pendingRequests.length > 0 ? 'text-amber-500' : 'text-muted-foreground'}`} />
          </CardHeader>
          <CardContent className="relative z-10">
            <div className={`text-3xl font-bold ${pendingRequests.length > 0 ? 'text-amber-500' : 'text-foreground'}`}>{pendingRequests.length}</div>
            <p className="text-xs text-muted-foreground mt-1">Requires immediate attention</p>
          </CardContent>
        </Card>

        <Card className="bg-card/60 backdrop-blur-sm border-border hover:border-primary/50 transition-colors">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-sm font-medium text-muted-foreground">Logins Today</CardTitle>
            <Users className="w-4 h-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-foreground">12</div>
            <p className="text-xs text-muted-foreground mt-1">Across 4 distinct clusters</p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Areas */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-7 mt-8">
        
        {/* Pending Requests Queue */}
        <Card className="col-span-4 bg-card/60 border-border backdrop-blur-sm">
          <CardHeader>
            <CardTitle>Approval Queue</CardTitle>
            <CardDescription>
              Review and manage infrastructure access requests.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {pendingRequests.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground text-sm">No pending requests.</div>
              ) : pendingRequests.map(req => (
                <div key={req.id} className="flex items-center justify-between p-4 border rounded-xl hover:bg-muted/30 transition-colors bg-background/50">
                  <div className="flex gap-4 items-center">
                    <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold">
                      {req.user_id.slice(0, 2).toUpperCase()}
                    </div>
                    <div>
                      <h4 className="font-semibold text-sm flex items-center gap-2">
                        {req.user?.name || "Dev User"} 
                        <span className="text-muted-foreground font-normal">requested</span>
                        <span className="text-primary">{req.server?.hostname || req.server_id}</span>
                      </h4>
                      <div className="flex flex-col gap-1 mt-1">
                        <p className="text-xs text-muted-foreground flex items-center gap-2">
                          <Clock className="w-3 h-3" /> {req.duration}
                          {req.sudo && <span className="text-destructive flex items-center gap-1 ml-2"><ShieldAlert className="w-3 h-3"/> Sudo</span>}
                          {req.requested_path && <span className="text-blue-400 flex items-center gap-1 ml-2">Path: {req.requested_path}</span>}
                          {req.requested_services && <span className="text-emerald-400 flex items-center gap-1 ml-2">Svc: {req.requested_services}</span>}
                        </p>
                        <p className="text-[10px] text-muted-foreground/70 font-mono flex items-center gap-1">
                          <Key className="w-3 h-3" /> {req.pub_key ? `${req.pub_key.substring(0, 25)}••••••••••••` : "No Key"}
                        </p>
                      </div>
                    </div>
                  </div>
                  <div className="flex flex-col gap-2 items-end">
                    <select 
                      className="text-[10px] bg-background border rounded px-1 h-6 focus:ring-0 outline-none"
                      id={`duration-${req.id}`}
                      defaultValue={req.duration}
                    >
                      <option value="5m">5m</option>
                      <option value="15m">15m</option>
                      <option value="30m">30m</option>
                      <option value="1h">1h</option>
                      <option value="4h">4h</option>
                      <option value="24h">24h</option>
                    </select>
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" className="h-8 gap-1 text-emerald-500 hover:text-emerald-400 hover:bg-emerald-500/10 border-emerald-500/20" onClick={() => {
                        const dur = (document.getElementById(`duration-${req.id}`) as HTMLSelectElement).value;
                        handleApprove(req.id, dur);
                      }}>
                        <CheckCircle2 className="w-4 h-4" /> Approve
                      </Button>
                      <Button variant="outline" size="sm" className="h-8 gap-1 text-destructive hover:bg-destructive/10 border-destructive/20" onClick={() => handleReject(req.id, true)}>
                        <XCircle className="w-4 h-4" /> Reject
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Active Sessions & Fleet Status */}
        <div className="col-span-3 space-y-6">
          <Card className="bg-card/60 border-border backdrop-blur-sm">
            <CardHeader>
              <CardTitle>Active Access</CardTitle>
              <CardDescription>Currently granted SSH sessions.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {requests.filter(r => r.status === 'approved' || r.status === 'active').length === 0 ? (
                  <div className="text-center py-4 text-muted-foreground text-xs">No active sessions.</div>
                ) : requests.filter(r => r.status === 'approved' || r.status === 'active').map(req => (
                  <div key={req.id} className="p-3 border rounded-lg bg-background/40 flex justify-between items-center">
                    <div>
                      <h4 className="font-medium text-xs text-primary">{req.user?.name || "User"}</h4>
                      <p className="text-[10px] text-muted-foreground mt-0.5">
                        {req.server?.hostname} • Expires {new Date(req.expires_at).toLocaleTimeString()}
                      </p>
                    </div>
                    <Button variant="ghost" size="sm" className="h-7 text-[10px] text-destructive hover:bg-destructive/10" onClick={() => handleRevoke(req.id)}>
                      Revoke
                    </Button>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card className="bg-card/60 border-border backdrop-blur-sm">
            <CardHeader>
              <CardTitle>Fleet Status</CardTitle>
              <CardDescription>
                Registered JIT Agents across your infrastructure.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {servers.length === 0 ? (
                   <div className="text-center py-4 text-muted-foreground text-sm">No servers registered.</div>
                ) : servers.map(srv => (
                  <div key={srv.id} className="flex items-center justify-between p-3 border rounded-lg bg-background/50">
                    <div>
                      <div className="flex items-center gap-2">
                        <div className={`w-2 h-2 rounded-full ${srv.status === 'online' ? 'bg-emerald-500 animate-pulse' : 'bg-muted-foreground'}`}></div>
                        <h4 className="font-medium text-sm">{srv.hostname}</h4>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1 font-mono">{srv.ip}</p>
                    </div>
                    <div className="flex gap-2">
                      {srv.tags?.map((t: any) => (
                         <Badge key={t.id} variant="secondary" className="text-[10px]">{t.k}: {t.v}</Badge>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
        
      </div>
    </>
  );
}
