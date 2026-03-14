"use client";

import { useEffect, useState } from "react";
import {
  Cloud,
  Clock,
  ShieldAlert,
  Send,
  AlertCircle,
  Trash2,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { apiFetch } from "@/lib/api";

const getCookie = (name: string): string | null => {
  if (typeof document === "undefined") return null;
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
};

interface Integration {
  id: string;
  name: string;
  provider: string;
  status: string;
  metadata?: string;
}

interface CloudRequest {
  id: string;
  user_id: string;
  target_group_name: string;
  duration_hours: number;
  reason: string;
  status: string;
  created_at: string;
  expires_at?: string;
  integration?: any;
  console_url?: string;
  temp_password?: string;
  temp_access_key?: string;
  temp_secret_key?: string;
}

export default function UserCloudPage() {
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [myRequests, setMyRequests] = useState<CloudRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [availableGroups, setAvailableGroups] = useState<
    { id: string; name: string }[]
  >([]);
  const [loadingGroups, setLoadingGroups] = useState(false);

  const [formData, setFormData] = useState({
    integration_id: "",
    target_group_id: "",
    target_group_name: "",
    duration_hours: 1,
    reason: "",
    requires_password: false,
    requires_keys: false,
  });

  const fetchData = async () => {
    try {
      setLoading(true);
      const userId = getCookie("jit_auth_id");

      const [intRes, reqRes] = await Promise.all([
        apiFetch("/cloud-integrations"),
        apiFetch("/cloud-requests"),
      ]);

      if (intRes.ok) {
        const ints = await intRes.json();
        // Only show active integrations to users
        const activeInts = ints.filter(
          (i: Integration) => i.status === "active",
        );
        setIntegrations(activeInts);
        if (activeInts.length > 0 && !formData.integration_id) {
          setFormData((prev) => ({
            ...prev,
            integration_id: activeInts[0].id,
          }));
        }
      }

      if (reqRes.ok) {
        const reqs = await reqRes.json();
        // Filter requests to only show the current user's requests
        if (userId) {
          setMyRequests(reqs.filter((r: CloudRequest) => r.user_id === userId));
        }
      }
    } catch (e) {
      console.error("Failed to fetch cloud data", e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (formData.integration_id) {
      setFormData((prev) => ({
        ...prev,
        target_group_id: "",
        target_group_name: "",
      }));
      fetchGroups(formData.integration_id);
    }
  }, [formData.integration_id]);

  const fetchGroups = async (integrationId: string) => {
    try {
      setLoadingGroups(true);
      const res = await apiFetch(`/cloud-integrations/${integrationId}/groups`);
      if (res.ok) {
        const data = await res.json();
        setAvailableGroups(data || []);
      } else {
        setAvailableGroups([]);
      }
    } catch (e) {
      console.error("Failed to fetch groups", e);
      setAvailableGroups([]);
    } finally {
      setLoadingGroups(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to cancel this request?")) return;
    try {
      const res = await apiFetch(`/cloud-requests/${id}`, {
        method: "DELETE",
      });
      if (res.ok) {
        fetchData();
      } else {
        const err = await res.json();
        alert(err.error || "Failed to cancel request.");
      }
    } catch (e) {
      console.error(e);
      alert("An error occurred while canceling the request.");
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const userId = getCookie("jit_auth_id");
    if (!userId) {
      alert("Authentication error: Could not identify user.");
      return;
    }

    if (
      !formData.integration_id ||
      !formData.target_group_id ||
      !formData.target_group_name ||
      !formData.reason
    ) {
      alert("Please fill in all required fields.");
      return;
    }

    try {
      const res = await apiFetch("/cloud-requests", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          integration_id: formData.integration_id,
          target_group_id: formData.target_group_id,
          target_group_name: formData.target_group_name,
          duration_hours: Number(formData.duration_hours),
          reason: formData.reason,
          requires_password: formData.requires_password,
          requires_keys: formData.requires_keys,
        }),
      });

      if (res.ok) {
        setFormData({
          ...formData,
          target_group_id: "",
          target_group_name: "",
          reason: "",
          requires_password: false,
          requires_keys: false,
        });
        fetchData();
        alert(
          "Cloud access request submitted successfully! Waiting for admin approval.",
        );
      } else {
        const err = await res.json();
        alert(err.error || "Failed to submit request.");
      }
    } catch (e) {
      console.error(e);
      alert("An error occurred while submitting the request.");
    }
  };

  return (
    <>
      <div className="mb-8">
        <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
          <Cloud className="w-6 h-6 text-primary" /> Cloud Access
        </h2>
        <p className="text-muted-foreground mt-1">
          Request Just-In-Time access to AWS, Google Cloud, and Azure
          environments.
        </p>
      </div>

      <div className="grid gap-8 lg:grid-cols-3">
        {/* Request Form */}
        <div className="lg:col-span-1">
          <Card className="bg-card/60 backdrop-blur-sm border-border">
            <CardHeader>
              <CardTitle>Request Access</CardTitle>
              <CardDescription>
                Ask for temporary access to a cloud identity group.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {integrations.length === 0 && !loading ? (
                <div className="text-sm text-amber-500 flex items-center gap-2 p-3 bg-amber-500/10 rounded-md border border-amber-500/20">
                  <AlertCircle className="w-4 h-4" />
                  No active cloud integrations are available. Contact your
                  administrator.
                </div>
              ) : (
                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-xs font-semibold text-muted-foreground uppercase">
                      Cloud Environment
                    </label>
                    <select
                      className="w-full bg-background border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                      value={formData.integration_id}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          integration_id: e.target.value,
                        })
                      }
                      disabled={integrations.length === 0}
                    >
                      {integrations.map((int) => (
                        <option key={int.id} value={int.id}>
                          {int.name} ({int.provider.toUpperCase()})
                        </option>
                      ))}
                    </select>
                  </div>

                  {loadingGroups ? (
                    <div className="py-4 text-center text-sm text-muted-foreground animate-pulse border rounded-md bg-muted/20">
                      Loading available groups...
                    </div>
                  ) : availableGroups.length > 0 ? (
                    <div className="space-y-2">
                      <label className="text-xs font-semibold text-muted-foreground uppercase">
                        Select Target Group
                      </label>
                      <select
                        className="w-full bg-background border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                        value={formData.target_group_id}
                        onChange={(e) => {
                          const selected = availableGroups.find(
                            (g) => g.id === e.target.value,
                          );
                          setFormData({
                            ...formData,
                            target_group_id: e.target.value,
                            target_group_name: selected
                              ? selected.name
                              : e.target.value,
                          });
                        }}
                        required
                      >
                        <option value="">-- Select a Group --</option>
                        {availableGroups.map((g) => (
                          <option key={g.id} value={g.id}>
                            {g.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  ) : (
                    <>
                      <div className="space-y-2">
                        <label className="text-xs font-semibold text-muted-foreground uppercase">
                          Target Group Name
                        </label>
                        <input
                          className="w-full bg-background border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                          placeholder="e.g. SRE-Production-Admin"
                          value={formData.target_group_name}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              target_group_name: e.target.value,
                            })
                          }
                          required
                        />
                      </div>

                      <div className="space-y-2">
                        <label className="text-xs font-semibold text-muted-foreground uppercase">
                          Target Group ID
                        </label>
                        <input
                          className="w-full bg-background border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                          placeholder="e.g. 1a2b3c4d-5e6f... (AWS ID) or Group Name"
                          value={formData.target_group_id}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              target_group_id: e.target.value,
                            })
                          }
                          required
                        />
                        <p className="text-[10px] text-muted-foreground">
                          For Identity Center/Azure, enter the exact Object ID.
                          For AWS IAM (Legacy), enter the Group Name (not the
                          ARN).
                        </p>
                      </div>
                    </>
                  )}

                  <div className="space-y-2">
                    <label className="text-xs font-semibold text-muted-foreground uppercase">
                      Duration
                    </label>
                    <select
                      className="w-full bg-background border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary"
                      value={formData.duration_hours}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          duration_hours: Number(e.target.value),
                        })
                      }
                    >
                      <option value={1}>1 Hour</option>
                      <option value={4}>4 Hours</option>
                      <option value={8}>8 Hours</option>
                      <option value={24}>24 Hours</option>
                    </select>
                  </div>

                  <div className="space-y-2">
                    <label className="text-xs font-semibold text-muted-foreground uppercase">
                      Business Justification
                    </label>
                    <textarea
                      className="w-full bg-background border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary min-h-[80px]"
                      placeholder="Why do you need this access?"
                      value={formData.reason}
                      onChange={(e) =>
                        setFormData({ ...formData, reason: e.target.value })
                      }
                      required
                    />
                  </div>

                  {integrations.find((i) => i.id === formData.integration_id)
                    ?.provider === "aws-iam" && (
                    <div className="space-y-3 p-3 bg-muted/20 border rounded-md">
                      <label className="text-xs font-semibold text-muted-foreground uppercase">
                        Legacy IAM Options
                      </label>
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id="req_pass"
                          checked={formData.requires_password}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              requires_password: e.target.checked,
                            })
                          }
                          className="rounded border-input"
                        />
                        <label htmlFor="req_pass" className="text-sm">
                          Generate Console Password
                        </label>
                      </div>
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id="req_keys"
                          checked={formData.requires_keys}
                          onChange={(e) =>
                            setFormData({
                              ...formData,
                              requires_keys: e.target.checked,
                            })
                          }
                          className="rounded border-input"
                        />
                        <label htmlFor="req_keys" className="text-sm">
                          Generate Programmatic Access Keys
                        </label>
                      </div>
                    </div>
                  )}

                  <Button
                    type="submit"
                    className="w-full gap-2 mt-4"
                    disabled={integrations.length === 0 || loading}
                  >
                    <Send className="w-4 h-4" /> Submit Request
                  </Button>
                </form>
              )}
            </CardContent>
          </Card>
        </div>

        {/* My Requests History */}
        <div className="lg:col-span-2 space-y-6">
          <Card className="bg-card/60 backdrop-blur-sm border-border">
            <CardHeader>
              <CardTitle>My Cloud Requests</CardTitle>
              <CardDescription>
                Track your active and previous cloud access requests.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loading && myRequests.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  Loading requests...
                </div>
              ) : myRequests.length === 0 ? (
                <div className="text-center py-12 text-muted-foreground bg-muted/20 rounded-lg border border-dashed">
                  <Cloud className="w-8 h-8 mx-auto mb-3 opacity-20" />
                  You haven&apos;t made any cloud access requests yet.
                </div>
              ) : (
                <div className="space-y-4">
                  {myRequests
                    .sort(
                      (a, b) =>
                        new Date(b.created_at).getTime() -
                        new Date(a.created_at).getTime(),
                    )
                    .map((req) => (
                      <div
                        key={req.id}
                        className="p-4 border rounded-xl hover:bg-muted/30 transition-colors bg-background/50 flex flex-col gap-4"
                      >
                        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                          <div className="flex gap-4 items-start">
                            <div
                              className={`w-10 h-10 rounded-full flex items-center justify-center shrink-0 ${
                                req.status === "active"
                                  ? "bg-emerald-500/10 text-emerald-500"
                                  : req.status === "pending"
                                    ? "bg-amber-500/10 text-amber-500"
                                    : req.status === "revoked"
                                      ? "bg-destructive/10 text-destructive"
                                      : "bg-muted text-muted-foreground"
                              }`}
                            >
                              <Cloud className="w-5 h-5" />
                            </div>

                            <div>
                              <h4 className="font-semibold text-sm flex items-center gap-2">
                                {req.target_group_name}
                                <Badge
                                  variant="outline"
                                  className="text-[10px] uppercase tracking-wider py-0 font-normal"
                                >
                                  {req.integration?.provider || "Cloud"}
                                </Badge>
                              </h4>
                              <div className="flex flex-col gap-1 mt-1">
                                <p className="text-xs text-muted-foreground flex items-center gap-2">
                                  <ShieldAlert className="w-3 h-3" />{" "}
                                  {req.integration?.name ||
                                    "Unknown Environment"}
                                </p>
                                <p className="text-xs text-muted-foreground flex items-center gap-2">
                                  <Clock className="w-3 h-3" /> Requested{" "}
                                  {req.duration_hours} hours
                                </p>
                                <p className="text-[10px] text-muted-foreground/70 italic max-w-sm truncate mt-1">
                                  &quot;{req.reason}&quot;
                                </p>
                              </div>
                            </div>
                          </div>

                          <div className="flex flex-col items-start md:items-end gap-2 border-t md:border-t-0 pt-3 md:pt-0">
                            <Badge
                              variant={
                                req.status === "active"
                                  ? "default"
                                  : req.status === "pending"
                                    ? "secondary"
                                    : req.status === "revoked"
                                      ? "destructive"
                                      : "outline"
                              }
                            >
                              {req.status}
                            </Badge>

                            {req.status === "active" && req.expires_at && (
                              <div className="text-[10px] text-emerald-500 font-medium bg-emerald-500/10 px-2 py-1 rounded">
                                Expires:{" "}
                                {new Date(req.expires_at).toLocaleString()}
                              </div>
                            )}

                            <div className="text-[10px] text-muted-foreground">
                              Created:{" "}
                              {new Date(req.created_at).toLocaleDateString()}
                            </div>

                            {req.status === "pending" && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-6 text-[10px] text-destructive hover:text-destructive hover:bg-destructive/10 mt-1 w-full justify-start md:justify-end px-2"
                                onClick={() => handleDelete(req.id)}
                              >
                                <Trash2 className="w-3 h-3 mr-1" /> Cancel
                              </Button>
                            )}
                          </div>
                        </div>

                        {req.status === "active" &&
                          (req.temp_password || req.temp_access_key) &&
                          req.integration?.provider !== "aws" && (
                            <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded-md text-xs space-y-1 mt-2">
                              <p className="font-semibold text-amber-500 mb-2">
                                Temporary Credentials Provisioned
                              </p>
                              {req.temp_password && (
                                <p>
                                  <span className="text-muted-foreground">
                                    Console Login:
                                  </span>{" "}
                                  <code className="bg-background px-1.5 py-0.5 rounded text-amber-500 font-mono ml-1">
                                    {req.temp_password}
                                  </code>
                                </p>
                              )}
                              {req.console_url ? (
                                <p className="pt-2">
                                  <a
                                    href={req.console_url}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-blue-500 hover:underline font-medium inline-flex items-center gap-1"
                                  >
                                    Open AWS Console Login &rarr;
                                  </a>
                                </p>
                              ) : req.temp_password &&
                                req.temp_password.includes("http") ? (
                                <p className="pt-2">
                                  <a
                                    href={
                                      (req.temp_password.match(
                                        /https?:\/\/[^\s)]+/,
                                      ) || [""])[0]
                                    }
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-blue-500 hover:underline font-medium inline-flex items-center gap-1"
                                  >
                                    Open AWS Console Login &rarr;
                                  </a>
                                </p>
                              ) : (
                                req.integration?.metadata &&
                                (() => {
                                  try {
                                    const meta = JSON.parse(
                                      req.integration.metadata,
                                    );
                                    if (meta.account_id) {
                                      return (
                                        <p className="pt-2">
                                          <a
                                            href={`https://${meta.account_id}.signin.aws.amazon.com/console`}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-blue-500 hover:underline font-medium inline-flex items-center gap-1"
                                          >
                                            Open AWS Console Login &rarr;
                                          </a>
                                        </p>
                                      );
                                    }
                                  } catch (e) {
                                    return null;
                                  }
                                  return null;
                                })()
                              )}
                              {req.temp_access_key && (
                                <>
                                  <p>
                                    <span className="text-muted-foreground">
                                      Access Key ID:
                                    </span>{" "}
                                    <code className="bg-background px-1.5 py-0.5 rounded text-amber-500 font-mono ml-1">
                                      {req.temp_access_key}
                                    </code>
                                  </p>
                                  <p>
                                    <span className="text-muted-foreground">
                                      Secret Access Key:
                                    </span>{" "}
                                    <code className="bg-background px-1.5 py-0.5 rounded text-amber-500 font-mono ml-1">
                                      {req.temp_secret_key}
                                    </code>
                                  </p>
                                </>
                              )}
                            </div>
                          )}

                        {req.status === "active" &&
                          req.integration?.provider === "aws" && (
                            <div className="p-3 bg-blue-500/10 border border-blue-500/20 rounded-md text-xs space-y-2 mt-2">
                              <p className="font-semibold text-blue-500">
                                AWS SSO Access Granted
                              </p>
                              <p className="text-muted-foreground">
                                You have been temporarily added to the requested
                                Identity Center group. You can now log into your
                                AWS SSO Portal.
                              </p>
                              {(req.console_url ||
                                (req.temp_password &&
                                  req.temp_password.includes("http"))) && (
                                <div className="mt-2">
                                  <a
                                    href={
                                      req.console_url ||
                                      (req.temp_password?.match(
                                        /https?:\/\/[^\s)]+/,
                                      ) || [""])[0]
                                    }
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-blue-500 hover:underline font-medium inline-flex items-center gap-1"
                                  >
                                    Open AWS SSO Portal &rarr;
                                  </a>
                                </div>
                              )}
                            </div>
                          )}
                      </div>
                    ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </>
  );
}
