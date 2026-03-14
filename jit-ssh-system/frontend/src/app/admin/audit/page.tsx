"use client";

import { useEffect, useState } from "react";
import { ShieldCheck, Calendar, Activity, RefreshCw } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { apiFetch } from "@/lib/api";

export default function AuditLogsPage() {
  const [logs, setLogs] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = async () => {
    try {
      setLoading(true);
      const res = await apiFetch("/logs");
      if (res.ok) setLogs(await res.json());
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return (
    <>
      <div className="flex justify-between items-center bg-card/60 p-6 rounded-xl border border-border backdrop-blur-sm mb-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Audit Logs</h2>
          <p className="text-muted-foreground mt-1">Immutable record of all JIT access requests, approvals, and system events.</p>
        </div>
        <Button onClick={fetchData} variant="outline" size="icon" disabled={loading}>
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      <Card className="bg-card/60 border-border backdrop-blur-sm">
        <CardHeader>
          <CardTitle>System Activity</CardTitle>
          <CardDescription>
            Chronological log of events across the control plane.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-8 text-muted-foreground animate-pulse">Loading audit trail...</div>
          ) : logs.length === 0 ? (
            <div className="text-center py-12 border-2 border-dashed rounded-xl border-border">
              <ShieldCheck className="w-12 h-12 text-muted-foreground mx-auto mb-4 opacity-50" />
              <h3 className="text-lg font-medium">No Logs Found</h3>
              <p className="text-sm text-muted-foreground">The audit trail is currently empty.</p>
            </div>
          ) : (
            <div className="rounded-md border border-border overflow-hidden">
              <table className="w-full text-sm text-left">
                <thead className="text-xs text-muted-foreground uppercase bg-muted/50 border-b border-border">
                  <tr>
                    <th className="px-6 py-3"><Calendar className="w-4 h-4 inline-block mr-1"/> Timestamp</th>
                    <th className="px-6 py-3">User ID</th>
                    <th className="px-6 py-3">Target Server</th>
                    <th className="px-6 py-3">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log, idx) => (
                    <tr key={idx} className="border-b border-border bg-background/50 hover:bg-muted/50 transition-colors">
                      <td className="px-6 py-4 font-mono text-xs text-muted-foreground whitespace-nowrap">
                        {new Date(log.timestamp).toLocaleString()}
                      </td>
                      <td className="px-6 py-4 font-mono text-[11px] text-muted-foreground">
                        {log.user_id}
                      </td>
                      <td className="px-6 py-4 font-mono text-[11px] text-primary">
                        {log.server_id || log.server?.id || "System"}
                      </td>
                      <td className="px-6 py-4">
                        <Badge variant="outline" className="font-normal border-primary/20 bg-primary/5 text-primary">
                          <Activity className="w-3 h-3 mr-1" /> {log.action}
                        </Badge>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </>
  );
}
