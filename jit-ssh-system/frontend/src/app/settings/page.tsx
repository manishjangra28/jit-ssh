"use client";

import { useEffect, useState } from "react";
import { Key, Save, Check, Eye, EyeOff, Lock, ShieldCheck } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

export default function UserSettingsPage() {
  const [pubKey, setPubKey] = useState("");
  const [saved, setSaved] = useState(false);
  const [showKey, setShowKey] = useState(false);

  // Change password state
  const [currentPwd, setCurrentPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [pwdLoading, setPwdLoading] = useState(false);
  const [pwdSuccess, setPwdSuccess] = useState("");
  const [pwdError, setPwdError] = useState("");

  // Get current user from cookie
  const getUserId = () => {
    const cookie = document.cookie.split("; ").find(r => r.startsWith("jit_auth_id="));
    return cookie?.split("=")[1] || "";
  };
  const getUserEmail = () => {
    const c = document.cookie.split("; ").find(r => r.startsWith("jit_auth_email="));
    return decodeURIComponent(c?.split("=")[1] || "");
  };

  useEffect(() => {
    const savedKey = localStorage.getItem("jit_saved_pub_key");
    if (savedKey) setPubKey(savedKey);
  }, []);

  const handleSave = () => {
    localStorage.setItem("jit_saved_pub_key", pubKey);
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwdError("");
    setPwdSuccess("");

    if (newPwd.length < 6) {
      setPwdError("New password must be at least 6 characters.");
      return;
    }
    if (newPwd !== confirmPwd) {
      setPwdError("New passwords do not match.");
      return;
    }

    const userId = getUserId();
    if (!userId) {
      setPwdError("Could not determine logged-in user. Please log in again.");
      return;
    }

    try {
      setPwdLoading(true);
      // First verify current password by calling login
      const email = getUserEmail();
      const verifyRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password: currentPwd }),
      });

      if (!verifyRes.ok) {
        setPwdError("Current password is incorrect.");
        return;
      }

      // Now set the new password
      const setRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"}/auth/set-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: userId, password: newPwd }),
      });

      if (setRes.ok) {
        setPwdSuccess("✅ Password changed successfully!");
        setCurrentPwd("");
        setNewPwd("");
        setConfirmPwd("");
      } else {
        const d = await setRes.json();
        setPwdError(d.error || "Failed to change password.");
      }
    } catch (err) {
      setPwdError("Network error — please try again.");
    } finally {
      setPwdLoading(false);
    }
  };

  const inputClass = "flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary pr-10";

  return (
    <>
      <div className="bg-card/60 p-6 rounded-xl border border-border backdrop-blur-sm mb-6">
        <h2 className="text-2xl font-bold tracking-tight">My Profile Settings</h2>
        <p className="text-muted-foreground mt-1">Configure your JIT access preferences.</p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* SSH Key card */}
        <Card className="bg-card/60 border-border backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Key className="w-5 h-5 text-primary" /> Default SSH Key
            </CardTitle>
            <CardDescription>
              Save your public Ed25519 or RSA key locally. It will auto-populate when you request access.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <label className="text-sm font-medium leading-none">Public SSH Key</label>
                <Button type="button" variant="ghost" size="sm" className="h-4 text-xs text-muted-foreground p-0 gap-1 hover:bg-transparent" onClick={() => setShowKey(!showKey)}>
                  {showKey ? <><EyeOff className="w-3 h-3"/>Hide</> : <><Eye className="w-3 h-3"/>Reveal</>}
                </Button>
              </div>
              <div className="relative">
                <textarea
                  className={`flex min-h-[120px] w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm font-mono ${!showKey ? 'text-transparent blur-[4px]' : ''}`}
                  placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
                  value={pubKey}
                  onChange={(e) => setPubKey(e.target.value)}
                />
                {!showKey && pubKey && (
                  <div className="absolute inset-0 flex p-3 pointer-events-none text-muted-foreground text-sm font-mono truncate">
                    {pubKey.substring(0, 16)}•••••••••••••••••••••
                  </div>
                )}
              </div>
              <p className="text-xs text-muted-foreground">Stored securely in browser local storage only.</p>
            </div>
          </CardContent>
          <CardFooter>
            <Button onClick={handleSave} className={`gap-2 ${saved ? 'bg-emerald-500 hover:bg-emerald-600 text-white' : ''}`} variant="default">
              {saved ? <><Check className="w-4 h-4"/>Saved</> : <><Save className="w-4 h-4"/>Save Key Locally</>}
            </Button>
          </CardFooter>
        </Card>

        {/* Change Password Card */}
        <Card className="bg-card/60 border-border backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Lock className="w-5 h-5 text-amber-500" /> Change My Password
            </CardTitle>
            <CardDescription>
              Update your login password. You'll need your current password to confirm.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleChangePassword} className="space-y-4">
              {/* Current password */}
              <div className="space-y-1">
                <label className="text-sm font-medium">Current Password</label>
                <div className="relative">
                  <input
                    type={showCurrent ? "text" : "password"}
                    required
                    className={inputClass}
                    placeholder="Your current password"
                    value={currentPwd}
                    onChange={e => setCurrentPwd(e.target.value)}
                  />
                  <button type="button" className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground" onClick={() => setShowCurrent(!showCurrent)}>
                    {showCurrent ? <EyeOff className="w-4 h-4"/> : <Eye className="w-4 h-4"/>}
                  </button>
                </div>
              </div>

              {/* New password */}
              <div className="space-y-1">
                <label className="text-sm font-medium">New Password</label>
                <div className="relative">
                  <input
                    type={showNew ? "text" : "password"}
                    required
                    className={inputClass}
                    placeholder="At least 6 characters"
                    value={newPwd}
                    onChange={e => setNewPwd(e.target.value)}
                  />
                  <button type="button" className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground" onClick={() => setShowNew(!showNew)}>
                    {showNew ? <EyeOff className="w-4 h-4"/> : <Eye className="w-4 h-4"/>}
                  </button>
                </div>
              </div>

              {/* Confirm */}
              <div className="space-y-1">
                <label className="text-sm font-medium">Confirm New Password</label>
                <input
                  type="password"
                  required
                  className="flex h-10 w-full rounded-md border border-input bg-background/50 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  placeholder="Repeat new password"
                  value={confirmPwd}
                  onChange={e => setConfirmPwd(e.target.value)}
                />
              </div>

              {pwdError && <div className="text-sm text-red-400 bg-red-400/10 border border-red-400/20 rounded-md px-3 py-2">{pwdError}</div>}
              {pwdSuccess && <div className="text-sm text-emerald-400 bg-emerald-400/10 border border-emerald-400/20 rounded-md px-3 py-2 flex items-center gap-2"><ShieldCheck className="w-4 h-4"/>{pwdSuccess}</div>}

              <Button type="submit" className="w-full gap-2" disabled={pwdLoading}>
                <Lock className="w-4 h-4"/>
                {pwdLoading ? "Updating..." : "Change Password"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </>
  );
}
