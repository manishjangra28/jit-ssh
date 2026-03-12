"use client";

import { useEffect, useState } from "react";
import { Plus, Trash2, KeyRound, Copy, Check, ShieldAlert, Clock, Info } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface AgentToken {
  id: string;
  label: string;
  server_id?: string;
  created_at: string;
  last_used_at?: string;
}

export default function AgentTokensPage() {
  const [tokens, setTokens] = useState<AgentToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newTokenLabel, setNewTokenLabel] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [revealedToken, setRevealedToken] = useState<{ label: string; token: string } | null>(null);
  const [copied, setCopied] = useState(false);

  const getApiUrl = () => {
    if (typeof window !== "undefined" && window.location.hostname !== "localhost") {
       // If running in docker/production, we might need a dynamic URL
       // but for now let's stick to the env or default
    }
    return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';
  };

  const fetchTokens = async () => {
    try {
      setLoading(true);
      const res = await fetch(`${getApiUrl()}/agent-tokens`);
      if (res.ok) {
        const data = await res.json();
        setTokens(Array.isArray(data) ? data : []);
      }
    } catch (e) {
      console.error("Failed to fetch tokens:", e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTokens();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTokenLabel) return;
    
    try {
      setSubmitting(true);
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'}/agent-tokens`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label: newTokenLabel })
      });
      
      if (res.ok) {
        const data = await res.json();
        setRevealedToken({ label: data.label, token: data.token });
        setNewTokenLabel("");
        setShowCreate(false);
        fetchTokens();
      }
    } catch (e) {
      console.error("Failed to create token:", e);
    } finally {
      setSubmitting(false);
    }
  };

  const handleRevoke = async (id: string) => {
    if (!confirm("Are you sure you want to revoke this token? Any agent using it will lose access immediately.")) return;
    
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'}/agent-tokens/${id}`, {
        method: "DELETE"
      });
      if (res.ok) {
        fetchTokens();
      }
    } catch (e) {
      console.error("Failed to revoke token:", e);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center bg-card/60 p-6 rounded-xl border border-border backdrop-blur-sm">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Agent Authentication Tokens</h2>
          <p className="text-muted-foreground mt-1">Generate pre-shared secrets to secure communication between your agents and the control plane.</p>
        </div>
        <Button onClick={() => setShowCreate(true)} className="gap-2">
          <Plus className="w-4 h-4" /> Generate New Token
        </Button>
      </div>

      {revealedToken && (
        <Card className="border-blue-500/50 bg-blue-500/5 backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2 text-blue-400">
              <ShieldAlert className="w-5 h-5" /> New Token Generated: {revealedToken.label}
            </CardTitle>
            <CardDescription className="text-blue-400/80">
              Copy this token now. It will NOT be shown again for security reasons.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-3 bg-background/50 p-4 rounded-lg border border-blue-500/30">
              <code className="flex-1 font-mono text-sm break-all select-all">{revealedToken.token}</code>
              <Button size="sm" variant="outline" className="gap-2 border-blue-500/30 hover:bg-blue-500/10" onClick={() => copyToClipboard(revealedToken.token)}>
                {copied ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4" />}
                {copied ? "Copied!" : "Copy Token"}
              </Button>
            </div>
          </CardContent>
          <CardFooter>
            <Button size="sm" variant="ghost" onClick={() => setRevealedToken(null)} className="text-blue-400 hover:text-blue-300 hover:bg-blue-500/10">
              I've safely stored this token
            </Button>
          </CardFooter>
        </Card>
      )}

      {showCreate && !revealedToken && (
        <Card className="bg-card/60 border-border backdrop-blur-sm">
          <form onSubmit={handleCreate}>
            <CardHeader>
              <CardTitle>Generate Agent Token</CardTitle>
              <CardDescription>Give this token a descriptive label to identify the agent or group of agents using it.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <label className="text-sm font-medium">Token Label</label>
                <input 
                  autoFocus
                  className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm ring-offset-background outline-none focus:ring-2 focus:ring-primary"
                  placeholder="e.g. Production Web Farm, DB Cluster-1"
                  value={newTokenLabel}
                  onChange={e => setNewTokenLabel(e.target.value)}
                  required
                />
              </div>
            </CardContent>
            <CardFooter className="justify-end gap-3">
              <Button type="button" variant="ghost" onClick={() => setShowCreate(false)}>Cancel</Button>
              <Button type="submit" disabled={submitting}>{submitting ? 'Generating...' : 'Generate Token'}</Button>
            </CardFooter>
          </form>
        </Card>
      )}

      <Card className="bg-card/60 border-border backdrop-blur-sm">
        <CardHeader>
          <CardTitle>Active Tokens</CardTitle>
          <CardDescription>Currently valid tokens that allow agents to connect.</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-8 text-muted-foreground">Loading tokens...</div>
          ) : tokens.length === 0 ? (
            <div className="text-center py-12 border-2 border-dashed rounded-xl border-border">
              <KeyRound className="w-12 h-12 text-muted-foreground mx-auto mb-4 opacity-50" />
              <h3 className="text-lg font-medium text-muted-foreground">No Tokens Found</h3>
              <p className="text-sm text-muted-foreground">Generate a token to start connecting agents.</p>
            </div>
          ) : (
            <div className="rounded-md border border-border overflow-hidden">
              <table className="w-full text-sm text-left">
                <thead className="text-xs text-muted-foreground uppercase bg-muted/50 border-b border-border">
                  <tr>
                    <th className="px-6 py-3">Label</th>
                    <th className="px-6 py-3">Attached Server</th>
                    <th className="px-6 py-3 text-center">Last Active</th>
                    <th className="px-6 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {tokens.map((token) => (
                    <tr key={token.id} className="border-b border-border bg-background/50 hover:bg-muted/50 transition-colors">
                      <td className="px-6 py-4">
                        <div className="font-medium flex items-center gap-2">
                          <ShieldAlert className="w-3 h-3 text-blue-400" />
                          {token.label}
                        </div>
                        <div className="text-[10px] text-muted-foreground font-mono mt-1">{token.id}</div>
                      </td>
                      <td className="px-6 py-4">
                        {token.server_id ? (
                          <Badge variant="outline" className="font-mono text-[10px] bg-primary/5 text-primary border-primary/20">
                            {token.server_id}
                          </Badge>
                        ) : (
                          <span className="text-muted-foreground italic text-xs">Not used yet</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-center">
                        {token.last_used_at ? (
                          <div className="flex flex-col items-center">
                            <span className="text-xs">{new Date(token.last_used_at).toLocaleString()}</span>
                            <span className="text-[10px] text-muted-foreground flex items-center gap-1">
                              <Clock className="w-2 h-2" /> Active
                            </span>
                          </div>
                        ) : (
                          <span className="text-muted-foreground text-xs">Never used</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="text-muted-foreground hover:text-destructive transition-colors"
                          onClick={() => handleRevoke(token.id)}
                          title="Revoke Token"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <div className="bg-amber-500/5 border border-amber-500/20 rounded-lg p-4 flex gap-3">
        <Info className="w-5 h-5 text-amber-500 shrink-0" />
        <div className="text-sm text-amber-200/80">
          <p className="font-bold text-amber-500 mb-1">Security Best Practice</p>
          <p>Always use unique tokens per environment or cluster. If an agent machine is compromised, revoke its token immediately to block access to the JIT control plane.</p>
        </div>
      </div>
    </div>
  );
}
