"use client";

import { useEffect, useState } from "react";
import { Server, Tags, Plus, RefreshCw, Layers } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getApiUrl } from "@/lib/api";

export default function ServersPage() {
  const [servers, setServers] = useState<any[]>([]);
  const [teams, setTeams] = useState<any[]>([]);
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState<string | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [srvRes, tRes, uRes] = await Promise.all([
        fetch(`${getApiUrl()}/servers`),
        fetch(`${getApiUrl()}/teams`),
        fetch(`${getApiUrl()}/users`)
      ]);
      if (srvRes.ok) setServers(await srvRes.json());
      if (tRes.ok) setTeams(await tRes.json());
      if (uRes.ok) setUsers(await uRes.json());
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const changeServerTeam = async (serverId: string, newTeamId: string) => {
    setUpdating(serverId);
    try {
      const res = await fetch(`${getApiUrl()}/servers/${serverId}/team`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ team_id: newTeamId })
      });
      if (res.ok) fetchData();
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(null);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return (
    <>
      <div className="flex justify-between items-center bg-card/60 p-6 rounded-xl border border-border backdrop-blur-sm mb-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Servers & Clusters</h2>
          <p className="text-muted-foreground mt-1">Manage infrastructure nodes and logical cluster groups.</p>
        </div>
        <div className="flex gap-2">
          <Button onClick={fetchData} variant="outline" size="icon" disabled={loading}>
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
          <Button className="gap-2">
            <Plus className="w-4 h-4" /> Add Server
          </Button>
        </div>
      </div>

      <div className="grid gap-6">
        <Card className="bg-card/60 border-border backdrop-blur-sm">
          <CardHeader>
            <CardTitle>Registered Fleet</CardTitle>
            <CardDescription>
              All nodes actively registered via the JIT agent.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="text-center py-8 text-muted-foreground animate-pulse">Loading servers...</div>
            ) : servers.length === 0 ? (
              <div className="text-center py-12 border-2 border-dashed rounded-xl border-border">
                <Server className="w-12 h-12 text-muted-foreground mx-auto mb-4 opacity-50" />
                <h3 className="text-lg font-medium">No Servers Found</h3>
                <p className="text-sm text-muted-foreground">Agents haven't registered with the control plane yet.</p>
              </div>
            ) : (
              <div className="rounded-md border border-border overflow-hidden">
                <table className="w-full text-sm text-left">
                  <thead className="text-xs text-muted-foreground uppercase bg-muted/50 border-b border-border">
                    <tr>
                      <th className="px-6 py-3">Hostname</th>
                      <th className="px-6 py-3">Status</th>
                      <th className="px-6 py-3">Team Scope</th>
                      <th className="px-6 py-3">Team Approvers</th>
                      <th className="px-6 py-3">Tags</th>
                      <th className="px-6 py-3 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {servers.map((srv) => (
                      <tr key={srv.id} className="border-b border-border bg-background/50 hover:bg-muted/50 transition-colors">
                        <td className="px-6 py-4 font-medium flex items-center gap-3">
                          <div className={`w-2 h-2 rounded-full ${srv.status === 'online' ? 'bg-emerald-500' : 'bg-muted-foreground'}`}></div>
                          {srv.hostname}
                        </td>
                        <td className="px-6 py-4">
                          <Badge variant="secondary" className="font-normal border-primary/20 bg-primary/5 text-primary">
                            {srv.team?.name || 'Unassigned'}
                          </Badge>
                        </td>
                        <td className="px-6 py-4">
                           {(() => {
                              if (!srv.team_id) return <span className="text-xs text-muted-foreground italic">None (requires admin)</span>;
                              const approvers = users.filter(u => u.role === 'approver' && u.team_id === srv.team_id);
                              if (approvers.length === 0) return <span className="text-xs text-amber-500 italic">No approvers in this team</span>;
                              return (
                                <div className="flex flex-wrap gap-1">
                                  {approvers.map(a => (
                                    <Badge key={a.id} variant="outline" className="text-[10px] bg-amber-500/10 text-amber-500 border-amber-500/20">
                                      {a.name || a.email.split('@')[0]}
                                    </Badge>
                                  ))}
                                </div>
                              );
                           })()}
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex flex-wrap gap-1">
                            {srv.tags?.map((t: any) => (
                              <Badge key={t.id} variant="outline" className="text-[10px] bg-background">
                                <Tags className="w-3 h-3 mr-1" /> {t.k}: {t.v}
                              </Badge>
                            )) || <span className="text-muted-foreground text-xs italic">none</span>}
                          </div>
                        </td>
                        <td className="px-6 py-4 text-right">
                          <select 
                            className="mr-2 h-8 rounded-md border border-input bg-background/50 px-2 text-xs"
                            value={srv.team_id || ""}
                            onChange={(e) => changeServerTeam(srv.id, e.target.value)}
                            disabled={updating === srv.id}
                          >
                            <option value="">No Team Assigned</option>
                            {teams.map(t => (
                              <option key={t.id} value={t.id}>{t.name}</option>
                            ))}
                          </select>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* MOCK CLUSTER DATA BELOW */}
        <Card className="bg-card/60 border-border backdrop-blur-sm opacity-80">
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><Layers className="w-5 h-5"/> Logical Clusters</CardTitle>
            <CardDescription>
              Groups of servers organized for bulk access control.
            </CardDescription>
          </CardHeader>
          <CardContent>
             <div className="grid md:grid-cols-2 gap-4">
                <div className="p-4 border border-border rounded-xl bg-background/50 hover:border-primary/50 transition-colors">
                  <div className="flex justify-between items-start mb-2">
                    <h4 className="font-semibold">Production DB Cluster</h4>
                    <Badge variant="outline">2 Nodes</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground mb-4">Tag matching: `env:prod AND role:db`</p>
                  <Button variant="secondary" size="sm" className="w-full">View Servers</Button>
                </div>
                
                <div className="p-4 border border-border rounded-xl bg-background/50 hover:border-primary/50 transition-colors">
                  <div className="flex justify-between items-start mb-2">
                    <h4 className="font-semibold">K8s Worker Nodes</h4>
                    <Badge variant="outline">5 Nodes</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground mb-4">Tag matching: `env:prod AND k8s:worker`</p>
                  <Button variant="secondary" size="sm" className="w-full">View Servers</Button>
                </div>
             </div>
          </CardContent>
        </Card>
      </div>
    </>
  );
}
