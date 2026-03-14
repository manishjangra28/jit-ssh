"use client";

import { useState, useEffect } from "react";
import { Server, Key, Clock, ShieldCheck, ChevronRight, RefreshCw, Eye, EyeOff } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { apiFetch } from "@/lib/api";
import NotificationTray from "@/components/NotificationTray";

const getCookieInPortal = (name: string): string | null => {
  if (typeof document === "undefined") return null;
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
};

export default function DevPortal() {
  const [servers, setServers] = useState<any[]>([]);
  const [requests, setRequests] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  
  // Form State
  const [selectedServer, setSelectedServer] = useState("");
  const [duration, setDuration] = useState("1h");
  const [sudo, setSudo] = useState(false);
  const [requestedPath, setRequestedPath] = useState("");
  const [requestedServices, setRequestedServices] = useState("");
  const [pubKey, setPubKey] = useState("");
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [showKey, setShowKey] = useState(false);

  // User Info
  const userId = getCookieInPortal("jit_auth_id") || "11111111-1111-1111-1111-111111111111";

  useEffect(() => {
    const savedKey = localStorage.getItem("jit_saved_pub_key");
    if (savedKey) setPubKey(savedKey);
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [srvRes, reqRes] = await Promise.all([
        apiFetch("/servers"),
        apiFetch("/requests")
      ]);
      if (srvRes.ok) setServers(await srvRes.json());
      if (reqRes.ok) {
        const allReqs = await reqRes.json();
        setRequests(allReqs.filter((r: any) => r.user_id === userId));
      }
    } catch (e) {
      console.error("Failed to fetch data", e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    // Poll every 10 seconds for approval updates
    const intv = setInterval(fetchData, 10000);
    return () => clearInterval(intv);
  }, [userId]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    
    try {
      const res = await apiFetch("/requests", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          server_id: selectedServer,
          pub_key: pubKey,
          duration,
          sudo,
          requested_path: requestedPath,
          requested_services: requestedServices,
          reason
        })
      });
      
      // Auto-save the key for future requests
      localStorage.setItem("jit_saved_pub_key", pubKey);
      
      if (res.ok) {
        setSelectedServer("");
        setPubKey(localStorage.getItem("jit_saved_pub_key") || "");
        setReason("");
        setRequestedPath("");
        setRequestedServices("");
        fetchData();
      } else {
        const err = await res.json();
        alert("Error: " + (err.error || "Failed to submit request"));
      }
    } catch (e) {
      console.error(e);
      alert("Network error.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Access Control</h2>
          <p className="text-muted-foreground mt-1">Request and manage your just-in-time server access.</p>
        </div>
        <Button onClick={fetchData} variant="outline" size="icon" disabled={loading}>
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {/* Request Form */}
        <Card className="col-span-2 bg-card/60 backdrop-blur-sm border-border">
          <CardHeader>
            <CardTitle className="text-lg">New Access Request</CardTitle>
            <CardDescription>
              Submit a request for temporary SSH access. Awaiting manager approval.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              
              <div className="space-y-2">
                <label className="text-sm font-medium leading-none">Target Server</label>
                <div className="relative">
                  <Server className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <select 
                    className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 pl-10 text-sm ring-offset-background disabled:opacity-50 appearance-none"
                    value={selectedServer}
                    onChange={(e) => setSelectedServer(e.target.value)}
                    required
                  >
                    <option value="" disabled>Select a server...</option>
                    {servers.map(s => (
                      <option key={s.id} value={s.id}>
                        {s.hostname} ({s.ip}) - {s.status}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <label className="text-sm font-medium leading-none">Public SSH Key (Ed25519 or RSA)</label>
                  <Button type="button" variant="ghost" size="sm" className="h-4 text-xs text-muted-foreground p-0 gap-1 hover:bg-transparent" onClick={() => setShowKey(!showKey)}>
                    {showKey ? <><EyeOff className="w-3 h-3"/> Hide</> : <><Eye className="w-3 h-3"/> Reveal</>}
                  </Button>
                </div>
                <div className="relative">
                  <Key className="absolute left-3 top-3 h-4 w-4 text-muted-foreground z-10" />
                  <textarea 
                    className={`flex min-h-[80px] w-full rounded-md border border-input bg-background/50 px-3 py-2 pl-10 text-sm ring-offset-background disabled:opacity-50 font-mono ${!showKey ? 'text-transparent blur-[4px]' : ''}`}
                    placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
                    value={pubKey}
                    onChange={(e) => setPubKey(e.target.value)}
                    required
                  />
                  {!showKey && pubKey && (
                     <div className="absolute inset-0 flex items-center pl-10 pt-1 pointer-events-none text-muted-foreground text-sm font-mono truncate mr-4">
                        {pubKey.substring(0, 16)}•••••••••••••••••••••
                     </div>
                  )}
                </div>
              </div>

              <div className="flex gap-4">
                <div className="space-y-2 flex-1">
                  <label className="text-sm font-medium leading-none">Duration</label>
                  <div className="relative">
                    <Clock className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                    <select 
                      className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 pl-10 text-sm appearance-none"
                      value={duration}
                      onChange={(e) => setDuration(e.target.value)}
                    >
                      <option value="5m">5 Minutes</option>
                      <option value="15m">15 Minutes</option>
                      <option value="30m">30 Minutes</option>
                      <option value="1h">1 Hour (Recommended)</option>
                      <option value="4h">4 Hours</option>
                      <option value="8h">8 Hours (Full Shift)</option>
                      <option value="24h">24 Hours (Max)</option>
                    </select>
                  </div>
                </div>

                <div className="space-y-4 flex-1">
                  <label className="text-sm font-medium leading-none block">Privileges</label>
                  <div className="flex items-center space-x-2 border border-input p-2 rounded-md bg-background/50 h-10">
                    <input 
                      type="checkbox" 
                      id="sudo" 
                      className="w-4 h-4 accent-primary" 
                      checked={sudo}
                      onChange={(e) => setSudo(e.target.checked)}
                    />
                    <label htmlFor="sudo" className="text-sm font-medium leading-none flex items-center gap-1 cursor-pointer">
                      <ShieldCheck className="w-4 h-4 text-destructive" /> Request Root / Sudo
                    </label>
                  </div>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                   <label className="text-sm font-medium leading-none">Specific Path Access (Optional)</label>
                   <input 
                    type="text"
                    className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm"
                    placeholder="e.g. /var/log/nginx"
                    value={requestedPath}
                    onChange={(e) => setRequestedPath(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                   <label className="text-sm font-medium leading-none">Service Permissions (e.g. docker)</label>
                   <input 
                    type="text"
                    className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm"
                    placeholder="e.g. docker, www-data"
                    value={requestedServices}
                    onChange={(e) => setRequestedServices(e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium leading-none">Justification</label>
                <input 
                  type="text"
                  className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm"
                  placeholder="e.g. Debugging production issue INC-1234"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  required
                />
              </div>

              <Button type="submit" className="w-full" disabled={submitting}>
                {submitting ? "Submitting..." : "Submit Access Request"} <ChevronRight className="w-4 h-4 ml-2" />
              </Button>
            </form>
          </CardContent>
        </Card>

        {/* My Active Sessions */}
        <div className="space-y-6">
          <Card className="bg-card/60 backdrop-blur-sm border-border">
            <CardHeader>
              <CardTitle className="text-lg">My Requests</CardTitle>
              <CardDescription>Status of your JIT access requests.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {requests.length === 0 ? (
                   <p className="text-sm text-muted-foreground text-center py-4">No active requests found.</p>
                ) : requests.map((req) => (
                  <div key={req.id} className={`p-3 border rounded-lg ${req.status === 'approved' ? 'bg-primary/5 border-primary/20' : req.status === 'rejected' ? 'opacity-60 grayscale' : 'bg-background/50'}`}>
                    <div className="flex justify-between items-start">
                      <div>
                        <h4 className="font-medium text-sm text-primary">{req.server?.hostname || req.server_id}</h4>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {req.status === 'approved' ? `Expires at ${new Date(req.expires_at).toLocaleTimeString()}` : `Duration: ${req.duration}`}
                        </p>
                      </div>
                      <Badge variant={req.status === 'approved' ? 'success' : req.status === 'pending' ? 'default' : 'secondary'}>
                        {req.status}
                      </Badge>
                    </div>
                    {req.status === 'approved' && req.server && (
                      <div className="mt-3 p-2 bg-background/80 rounded border font-mono text-[10px] text-muted-foreground overflow-x-auto">
                        ssh -i ~/.ssh/id_ed25519 tempuser@{req.server.ip}
                      </div>
                    )}
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
