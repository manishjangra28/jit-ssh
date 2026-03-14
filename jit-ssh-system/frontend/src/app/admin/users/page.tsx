"use client";

import { useEffect, useState } from "react";
import { Users, UserPlus, Settings2, RefreshCw, ShieldAlert, BadgeCheck, KeyRound, Trash2, Power } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { apiFetch } from "@/lib/api";

export default function UsersPage() {
  const [users, setUsers] = useState<any[]>([]);
  const [teams, setTeams] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newEmail, setNewEmail] = useState("");
  const [newRole, setNewRole] = useState("developer");
  const [newTeamId, setNewTeamId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [editUser, setEditUser] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [setPwdUser, setSetPwdUser] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState("");
  const [pwdMsg, setPwdMsg] = useState("");
  const [createdPwd, setCreatedPwd] = useState<{email: string; password: string; isReset?: boolean} | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [uRes, tRes] = await Promise.all([
        apiFetch("/users"),
        apiFetch("/teams"),
      ]);
      if (uRes.ok) setUsers(await uRes.json());
      if (tRes.ok) setTeams(await tRes.json());
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const changeRole = async (userId: string, newRole: string) => {
    setUpdating(userId);
    try {
      const res = await apiFetch(`/users/${userId}/role`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role: newRole })
      });
      if (res.ok) fetchData();
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(null);
    }
  };

  const changeTeam = async (userId: string, newTeamId: string) => {
    setUpdating(userId);
    try {
      const res = await apiFetch(`/users/${userId}/role`, {
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

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim() || !newEmail.trim()) return;
    try {
      setSubmitting(true);
      const res = await apiFetch("/users", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: newName, email: newEmail, role: newRole, team_id: newTeamId })
      });
      if (res.ok) {
        const data = await res.json();
        setNewName("");
        setNewEmail("");
        setNewRole("developer");
        setNewTeamId("");
        setShowCreate(false);
        // Show the generated one-time password to admin
        if (data.temp_password) {
          setCreatedPwd({ email: newEmail, password: data.temp_password });
        }
        fetchData();
      }
    } catch (e) {
      console.error(e);
    } finally {
      setSubmitting(false);
    }
  };

  const saveName = async (userId: string) => {
    setUpdating(userId);
    try {
      const res = await apiFetch(`/users/${userId}/role`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: editName })
      });
      if (res.ok) {
         setEditUser(null);
         fetchData();
      }
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(null);
    }
  };

  const roleColors: Record<string, string> = {
    admin: "text-red-500 bg-red-500/10 border-red-500/20",
    approver: "text-amber-500 bg-amber-500/10 border-amber-500/20",
    developer: "text-primary bg-primary/10 border-primary/20",
  };

  const handleSetPassword = async (userId: string) => {
    if (newPassword.length < 6) { setPwdMsg("Password must be at least 6 characters"); return; }
    setPwdMsg("");
    try {
      setSubmitting(true);
      const res = await apiFetch("/auth/set-password", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: userId, password: newPassword })
      });
      if (res.ok) {
        setSetPwdUser(null);
        setNewPassword("");
        setPwdMsg("");
      } else {
        const d = await res.json();
        setPwdMsg(d.error || "Failed");
      }
    } catch (e) {
      setPwdMsg("Network error");
    } finally {
      setSubmitting(false);
    }
  };

  const resetPassword = async (userId: string, userName: string, userEmail: string) => {
    try {
      setUpdating(userId);
      const res = await apiFetch(`/auth/reset-password/${userId}`, {
        method: "POST"
      });
      if (res.ok) {
        const data = await res.json();
        setCreatedPwd({ email: userEmail, password: data.temp_password, isReset: true });
      }
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(null);
    }
  };

  const deleteUser = async (userId: string, userName: string) => {
    if (!confirm(`Are you sure you want to permanently delete "${userName || userId}"? This cannot be undone.`)) return;
    try {
      setUpdating(userId);
      const res = await apiFetch(`/users/${userId}`, {
        method: "DELETE"
      });
      if (res.ok) fetchData();
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(null);
    }
  };

  const toggleStatus = async (userId: string) => {
    try {
      setUpdating(userId);
      const res = await apiFetch(`/users/${userId}/status`, {
        method: "PUT"
      });
      if (res.ok) fetchData();
    } catch (e) {
      console.error(e);
    } finally {
      setUpdating(null);
    }
  };

  const approvers = users.filter(u => u.role === "approver");
  const teamMap: Record<string, string> = Object.fromEntries(teams.map(t => [t.id, t.name]));

  // Robust clipboard copy that works in non-HTTPS contexts
  const copyToClipboard = (text: string) => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(text);
      } else {
        // fallback for HTTP / iframe
        const el = document.createElement("textarea");
        el.value = text;
        el.style.position = "fixed";
        el.style.opacity = "0";
        document.body.appendChild(el);
        el.focus();
        el.select();
        document.execCommand("copy");
        document.body.removeChild(el);
      }
    } catch (e) { console.error(e); }
  };

  const downloadCredentials = (email: string, password: string) => {
    const content = `JIT SSH System - Login Credentials\n=================================\nEmail:    ${email}\nPassword: ${password}\n\nLogin URL: http://localhost:3000/login\n\nNOTE: This is a one-time password. Please log in and change your password immediately.`;
    const blob = new Blob([content], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `jit-credentials-${email.split("@")[0]}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <>
      {/* One-Time Password Reveal Modal */}
      {createdPwd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
          <div className="bg-card border border-border rounded-2xl shadow-2xl p-8 max-w-md w-full mx-4">
            <div className="flex flex-col items-center text-center mb-6">
              <div className="w-14 h-14 bg-emerald-500/10 rounded-full flex items-center justify-center mb-3">
                <KeyRound className="w-7 h-7 text-emerald-500" />
              </div>
              <h3 className="text-xl font-bold">{createdPwd.isReset ? "Password Reset! 🔄" : "User Created! 🎉"}</h3>
              <p className="text-sm text-muted-foreground mt-1">Share these credentials with the user. The password will <strong>NOT</strong> be shown again.</p>
            </div>
            <div className="bg-background/70 rounded-xl border border-border p-4 space-y-3 mb-4">
              <div>
                <p className="text-xs text-muted-foreground mb-1">Email</p>
                <div className="flex items-center gap-2">
                  <p className="font-mono text-sm font-medium flex-1">{createdPwd.email}</p>
                  <button className="text-xs px-2 py-1 rounded bg-muted hover:bg-primary/20 transition-colors" onClick={() => copyToClipboard(createdPwd.email)}>Copy</button>
                </div>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-1">One-Time Password</p>
                <div className="flex items-center gap-2">
                  <code className="font-mono text-lg font-bold tracking-widest text-emerald-400 flex-1 break-all">{createdPwd.password}</code>
                  <button className="text-xs px-2 py-1 rounded bg-muted hover:bg-primary/20 transition-colors" onClick={() => copyToClipboard(createdPwd.password)}>Copy</button>
                </div>
              </div>
            </div>
            <div className="flex gap-3 mb-4">
              <Button variant="outline" className="flex-1 gap-2" onClick={() => downloadCredentials(createdPwd.email, createdPwd.password)}>
                ⬇ Download Credentials
              </Button>
              <Button variant="outline" className="flex-1 gap-2" onClick={() => copyToClipboard(`Email: ${createdPwd.email}\nPassword: ${createdPwd.password}`)}>
                Copy Both
              </Button>
            </div>
            <Button className="w-full" onClick={() => setCreatedPwd(null)}>Got it, I've saved the credentials</Button>
          </div>
        </div>
      )}
      <div className="flex justify-between items-center bg-card/60 p-6 rounded-xl border border-border backdrop-blur-sm mb-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">User Management</h2>
          <p className="text-muted-foreground mt-1">Control Developer access, Approver delegation, and Admin permissions.</p>
        </div>
        <div className="flex gap-2">
          <Button onClick={fetchData} variant="outline" size="icon" disabled={loading}>
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          </Button>
          <Button className="gap-2" onClick={() => setShowCreate(!showCreate)}>
            <UserPlus className="w-4 h-4" /> {showCreate ? "Cancel" : "Invite User"}
          </Button>
        </div>
      </div>

      {showCreate && (
        <Card className="bg-card/80 border-primary/30 mb-6 backdrop-blur-sm">
          <form onSubmit={handleCreate}>
            <CardHeader className="pb-3">
              <CardTitle className="text-lg">Invite New User</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Full Name</label>
                  <input required className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm" placeholder="Alice Engineer" value={newName} onChange={e => setNewName(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Email Address</label>
                  <input required type="email" className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm" placeholder="alice@example.com" value={newEmail} onChange={e => setNewEmail(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">System Role</label>
                  <select className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm" value={newRole} onChange={e => setNewRole(e.target.value)}>
                    <option value="developer">Developer</option>
                    <option value="approver">Approver</option>
                    <option value="admin">Administrator</option>
                  </select>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">Assign Team</label>
                  <select className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm" value={newTeamId} onChange={e => setNewTeamId(e.target.value)}>
                    <option value="">No Team</option>
                    {teams.map((t: any) => (
                       <option key={t.id} value={t.id}>{t.name}</option>
                    ))}
                  </select>
                </div>
              </div>
            </CardContent>
            <CardContent className="flex justify-end gap-2 pt-0">
               <Button type="button" variant="ghost" onClick={() => setShowCreate(false)}>Cancel</Button>
               <Button type="submit" disabled={submitting}>{submitting ? 'Creating...' : 'Send Invite'}</Button>
            </CardContent>
          </form>
        </Card>
      )}

      <Card className="bg-card/60 border-border backdrop-blur-sm">
        <CardHeader>
          <CardTitle>System Accounts</CardTitle>
          <CardDescription>
            Engineers, automated service accounts, and administrators.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="text-center py-8 text-muted-foreground animate-pulse">Loading users...</div>
          ) : users.length === 0 ? (
            <div className="text-center py-12 border-2 border-dashed rounded-xl border-border">
              <Users className="w-12 h-12 text-muted-foreground mx-auto mb-4 opacity-50" />
              <h3 className="text-lg font-medium">No Users Found</h3>
              <p className="text-sm text-muted-foreground">This system has no active accounts.</p>
            </div>
          ) : (
            <div className="rounded-md border border-border overflow-hidden">
              <table className="w-full text-sm text-left">
                <thead className="text-xs text-muted-foreground uppercase bg-muted/50 border-b border-border">
                  <tr>
                    <th className="px-6 py-3">Account</th>
                    <th className="px-6 py-3">Status</th>
                    <th className="px-6 py-3">System Role</th>
                    <th className="px-6 py-3">Assigned Team</th>
                    <th className="px-6 py-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((u) => (
                    <tr key={u.id} className={`border-b border-border transition-colors ${u.status === 'inactive' ? 'opacity-60 bg-muted/20' : 'bg-background/50 hover:bg-muted/50'}`}>
                      <td className="px-6 py-4">
                        {editUser === u.id ? (
                          <div className="flex items-center gap-2">
                            <input autoFocus className="h-8 rounded-md border border-input bg-background px-2 text-xs w-32" value={editName} onChange={e => setEditName(e.target.value)} />
                            <Button size="sm" variant="secondary" className="h-8" onClick={() => saveName(u.id)} disabled={updating === u.id}>Save</Button>
                            <Button size="sm" variant="ghost" className="h-8 text-muted-foreground" onClick={() => setEditUser(null)}>Cancel</Button>
                          </div>
                        ) : (
                          <>
                            <div className="font-medium text-foreground cursor-pointer hover:text-primary flex items-center gap-2 group" onClick={() => {setEditUser(u.id); setEditName(u.name);}}>
                              {u.name || "Unnamed"} <Settings2 className="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity" />
                            </div>
                            <div className="text-xs text-muted-foreground">{u.email}</div>
                          </>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        <Badge variant="outline" className={`text-[10px] ${u.status === 'inactive' ? 'text-muted-foreground bg-muted/30 border-border' : 'text-emerald-500 bg-emerald-500/10 border-emerald-500/20'}`}>
                          {u.status === 'inactive' ? '◦ Inactive' : '● Active'}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">
                        <Badge variant="outline" className={`font-normal ${roleColors[u.role] || roleColors.developer}`}>
                          {u.role === 'admin' && <ShieldAlert className="w-3 h-3 mr-1 inline" />}
                          {u.role === 'approver' && <BadgeCheck className="w-3 h-3 mr-1 inline" />}
                          {u.role}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">
                        <Badge variant="secondary" className="font-normal border-primary/20 bg-primary/5 text-primary">
                          {u.team?.name || 'Unassigned'}
                        </Badge>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <select 
                          className="mr-2 h-8 rounded-md border border-input bg-background/50 px-2 text-xs"
                          value={u.team_id || ""}
                          onChange={(e) => changeTeam(u.id, e.target.value)}
                          disabled={updating === u.id}
                        >
                          <option value="">No Team</option>
                          {teams.map(t => (
                            <option key={t.id} value={t.id}>{t.name}</option>
                          ))}
                        </select>
                        <select 
                          className="mr-2 h-8 rounded-md border border-input bg-background/50 px-2 text-xs"
                          value={u.role}
                          onChange={(e) => changeRole(u.id, e.target.value)}
                          disabled={updating === u.id}
                        >
                          <option value="developer">Set as Developer</option>
                          <option value="approver">Set as Approver</option>
                          <option value="admin">Set as Admin</option>
                        </select>
                        <Button variant="ghost" size="icon" className="h-8 w-8 text-blue-500" onClick={() => { setSetPwdUser(u.id); setNewPassword(""); setPwdMsg(""); }} title="Set Password">
                          <KeyRound className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="ghost" size="sm"
                          className="h-8 text-xs text-orange-400 hover:text-orange-300 px-2"
                          onClick={() => resetPassword(u.id, u.name, u.email)}
                          disabled={updating === u.id}
                        >Reset Pwd</Button>
                        <Button
                          variant="ghost" size="icon"
                          className={`h-8 w-8 ${u.status === 'active' ? 'text-yellow-500 hover:text-yellow-400' : 'text-emerald-500 hover:text-emerald-400'}`}
                          onClick={() => toggleStatus(u.id)}
                          disabled={updating === u.id}
                          title={u.status === 'active' ? 'Deactivate user' : 'Activate user'}
                        >
                          <Power className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="ghost" size="icon"
                          className="h-8 w-8 text-red-500 hover:text-red-400"
                          onClick={() => deleteUser(u.id, u.name)}
                          disabled={updating === u.id}
                          title="Delete user permanently"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                        {setPwdUser === u.id && (
                          <div className="absolute right-0 top-8 z-50 bg-card border border-border rounded-lg p-3 shadow-lg w-64">
                            <p className="text-xs font-medium mb-2">Set Password for <strong>{u.name || u.email}</strong></p>
                            <input
                              type="password"
                              className="flex h-8 w-full rounded-md border border-input bg-background/50 px-2 text-xs mb-2"
                              placeholder="Min 6 characters"
                              value={newPassword}
                              onChange={e => setNewPassword(e.target.value)}
                            />
                            {pwdMsg && <p className="text-xs text-red-400 mb-2">{pwdMsg}</p>}
                            <div className="flex gap-2">
                              <Button size="sm" className="text-xs h-7" onClick={() => handleSetPassword(u.id)} disabled={submitting}>Save</Button>
                              <Button size="sm" variant="ghost" className="text-xs h-7" onClick={() => setSetPwdUser(null)}>Cancel</Button>
                            </div>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Approver Group Section */}
      <Card className="bg-card/60 border-border backdrop-blur-sm mt-6">
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-amber-500">
            <ShieldAlert className="w-5 h-5" /> Approver Group
          </CardTitle>
          <CardDescription>Users with the <strong>Approver</strong> role. Each approver can only approve requests for servers assigned to their team.</CardDescription>
        </CardHeader>
        <CardContent>
          {approvers.length === 0 ? (
            <div className="text-center py-8 border-2 border-dashed border-amber-500/20 rounded-xl">
              <ShieldAlert className="w-8 h-8 text-amber-500/40 mx-auto mb-2" />
              <p className="text-sm text-muted-foreground">No approvers yet. Set a user's role to <strong>Approver</strong> above.</p>
            </div>
          ) : (
            <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
              {approvers.map(a => (
                <div key={a.id} className="flex items-center gap-3 p-3 rounded-lg border border-amber-500/20 bg-amber-500/5">
                  <div className="w-9 h-9 rounded-full bg-amber-500/10 flex items-center justify-center shrink-0">
                    <BadgeCheck className="w-5 h-5 text-amber-500" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{a.name || a.email}</p>
                    <p className="text-xs text-muted-foreground truncate">{a.email}</p>
                    <p className="text-xs mt-0.5">
                      {a.team_id ? (
                        <span className="text-amber-400">Team: {teamMap[a.team_id] || "Unknown"}</span>
                      ) : (
                        <span className="text-red-400">⚠ No team assigned — cannot approve any server</span>
                      )}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </>
  );
}
