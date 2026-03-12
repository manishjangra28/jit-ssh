"use client";

import { Settings as SettingsIcon, ShieldAlert } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

export default function SettingsPage() {
  return (
    <>
      <div className="bg-card/60 p-6 rounded-xl border border-border backdrop-blur-sm mb-6">
        <h2 className="text-2xl font-bold tracking-tight">System Settings</h2>
        <p className="text-muted-foreground mt-1">Configure global JIT SSH parameters and default behaviors.</p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card className="bg-card/60 border-border backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2"><SettingsIcon className="w-5 h-5"/> Global Limits</CardTitle>
            <CardDescription>
              Set maximum durations and default rules for JIT access.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Maximum Approval Duration (Hours)</label>
              <input 
                type="number"
                disabled
                className="flex h-10 w-full rounded-md border border-input bg-muted/50 px-3 py-2 text-sm text-muted-foreground cursor-not-allowed"
                value="24"
              />
              <p className="text-xs text-muted-foreground">System hard-limit. Requires backend flag override to exceed.</p>
            </div>
          </CardContent>
          <CardFooter>
            <Button disabled>Save Changes</Button>
          </CardFooter>
        </Card>

        <Card className="bg-destructive/5 border-destructive/20 backdrop-blur-sm">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2 text-destructive"><ShieldAlert className="w-5 h-5"/> Danger Zone</CardTitle>
            <CardDescription>
              Critical system actions.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
             <p className="text-sm text-muted-foreground text-balance">Revoking all active sessions will immediately instruct all offline and online JIT agents to forcefully terminate user sessions and delete temporal accounts.</p>
          </CardContent>
          <CardFooter>
            <Button variant="destructive" onClick={() => alert("Not implemented in demo")}>Revoke All Active Sessions</Button>
          </CardFooter>
        </Card>
      </div>
    </>
  );
}
