"use client";

import { useEffect, useState } from "react";
import { Layers, Plus, RefreshCw, Users } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getApiUrl } from "@/lib/api";

export default function TeamsPage() {
  const [teams, setTeams] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [editTeamId, setEditTeamId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editDesc, setEditDesc] = useState("");

  const fetchData = async () => {
    try {
      setLoading(true);
      const res = await fetch(`${getApiUrl()}/teams`);
      if (res.ok) setTeams(await res.json());
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim()) return;
    try {
      setSubmitting(true);
      const res = await fetch(`${getApiUrl()}/teams`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: newName, description: newDesc })
      });
      if (res.ok) {
        setNewName("");
        setNewDesc("");
        setShowCreate(false);
        fetchData();
      }
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleEditTeam = async (id: string) => {
    if (!editName.trim()) return;
    try {
      setSubmitting(true);
      const res = await fetch(`${getApiUrl()}/teams/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: editName, description: editDesc })
      });
      if (res.ok) {
        setEditTeamId(null);
        fetchData();
      }
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <div className="flex justify-between items-center bg-card/60 p-6 rounded-xl border border-border backdrop-blur-sm mb-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Teams & Groups</h2>
          <p className="text-muted-foreground mt-1">Manage infrastructure access clusters and team structures.</p>
        </div>
        <div className="flex gap-2">
          <Button onClick={fetchData} variant="outline" size="icon" disabled={loading}>
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
          <Button className="gap-2" onClick={() => setShowCreate(!showCreate)}>
            <Plus className="w-4 h-4" /> {showCreate ? "Cancel" : "Create Team"}
          </Button>
        </div>
      </div>

      {showCreate && (
        <Card className="bg-card/80 border-primary/30 mb-6 backdrop-blur-sm">
          <form onSubmit={handleCreate}>
            <CardHeader className="pb-3">
              <CardTitle className="text-lg">Create New Team</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Team Name</label>
                <input required className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm" placeholder="e.g. SRE Core" value={newName} onChange={e => setNewName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Description</label>
                <textarea className="flex min-h-[60px] w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm" placeholder="Optional description..." value={newDesc} onChange={e => setNewDesc(e.target.value)} />
              </div>
            </CardContent>
            <CardContent className="flex justify-end gap-2 pt-0">
               <Button type="button" variant="ghost" onClick={() => setShowCreate(false)}>Cancel</Button>
               <Button type="submit" disabled={submitting}>{submitting ? 'Creating...' : 'Create Team'}</Button>
            </CardContent>
          </form>
        </Card>
      )}

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {loading ? (
             <div className="col-span-full text-center py-8 text-muted-foreground animate-pulse">Loading teams...</div>
        ) : teams.length === 0 ? (
          <Card className="col-span-full bg-card/60 border-border border-dashed backdrop-blur-sm">
            <CardContent className="flex flex-col items-center justify-center p-12">
               <Layers className="w-12 h-12 text-muted-foreground mb-4 opacity-50" />
               <h3 className="text-lg font-medium">No Teams Configured</h3>
               <p className="text-sm text-muted-foreground text-center mt-1">Create teams like "DevOps", "Support", or "Backend DBAs" to logically group user access.</p>
            </CardContent>
          </Card>
        ) : teams.map((team) => (
           <Card key={team.id} className="bg-card/60 border-border backdrop-blur-sm hover:border-primary/50 transition-colors">
              <CardHeader className="pb-3">
                <div className="flex justify-between items-start mb-2">
                  <div className="w-10 h-10 rounded-lg bg-indigo-500/10 flex items-center justify-center">
                    <Users className="w-5 h-5 text-indigo-500"/>
                  </div>
                  <Badge variant="outline" className="font-mono text-[10px] text-muted-foreground">ID: {team.id.slice(0, 8)}</Badge>
                </div>
                {editTeamId === team.id ? (
                    <div className="space-y-2 mt-2">
                       <input autoFocus className="flex h-8 w-full rounded-md border border-input bg-background/50 px-2 text-sm" value={editName} onChange={e => setEditName(e.target.value)} />
                       <textarea className="flex min-h-[40px] w-full rounded-md border border-input bg-background/50 px-2 py-1 text-xs" value={editDesc} onChange={e => setEditDesc(e.target.value)} />
                       <div className="flex gap-2">
                          <Button size="sm" onClick={() => handleEditTeam(team.id)} disabled={submitting}>Save</Button>
                          <Button size="sm" variant="ghost" onClick={() => setEditTeamId(null)}>Cancel</Button>
                       </div>
                    </div>
                ) : (
                    <>
                      <CardTitle className="text-lg">{team.name}</CardTitle>
                      <CardDescription className="line-clamp-2 h-10 mt-1">
                        {team.description || "No description provided."}
                      </CardDescription>
                    </>
                )}
              </CardHeader>
              <CardContent className="pt-2">
                 <div className="text-xs text-muted-foreground flex justify-between items-center bg-background/50 p-2 rounded-md border border-border">
                    <span>Managed Role: <strong>Team Owner</strong></span>
                    <Button variant="link" size="sm" className="h-auto p-0 text-xs" onClick={() => { setEditTeamId(team.id); setEditName(team.name); setEditDesc(team.description || ""); }}>Edit Settings</Button>
                 </div>
              </CardContent>
           </Card>
        ))}
      </div>
    </>
  );
}
