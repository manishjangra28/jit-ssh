"use client";

import { useEffect, useState } from "react";
import {
  Cloud,
  Plus,
  Trash2,
  Edit2,
  CheckCircle2,
  XCircle,
  RefreshCw,
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

interface Integration {
  id: string;
  name: string;
  provider: string;
  status: string;
  metadata?: string;
  created_at: string;
}

interface CloudRequest {
  id: string;
  user_id: string;
  user?: {
    name: string;
    email: string;
  };
  integration_id?: string;
  integration?: {
    id: string;
    provider: string;
    name: string;
    metadata?: string;
  };
  target_group_id: string;
  target_group_name: string;
  duration_hours: number;
  status: string;
  reason?: string;
  created_at: string;
}

export default function AdminCloudPage() {
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [requests, setRequests] = useState<CloudRequest[]>([]);
  const [loading, setLoading] = useState(true);

  // Form Modal State (Add/Edit)
  const [showFormModal, setShowFormModal] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: "",
    provider: "aws",
    credentials: "",
    metadata: "",
  });

  // Approval Modal State
  const [showApproveModal, setShowApproveModal] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState<CloudRequest | null>(
    null,
  );
  const [availableGroups, setAvailableGroups] = useState<
    { id: string; name: string }[]
  >([]);
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [overrideGroupId, setOverrideGroupId] = useState("");
  const [overrideGroupName, setOverrideGroupName] = useState("");

  const fetchData = async () => {
    try {
      setLoading(true);
      const [intRes, reqRes] = await Promise.all([
        apiFetch("/cloud-integrations"),
        apiFetch("/cloud-requests"),
      ]);
      if (intRes.ok) setIntegrations(await intRes.json());
      if (reqRes.ok) setRequests(await reqRes.json());
    } catch (e) {
      console.error("Failed to fetch data", e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const openAddModal = () => {
    setIsEditing(false);
    setEditingId(null);
    setFormData({ name: "", provider: "aws", credentials: "", metadata: "" });
    setShowFormModal(true);
  };

  const openEditModal = (integration: Integration) => {
    setIsEditing(true);
    setEditingId(integration.id);
    setFormData({
      name: integration.name,
      provider: integration.provider,
      credentials: "", // Blank by default, only updated if typed
      metadata: integration.metadata || "",
    });
    setShowFormModal(true);
  };

  const handleSaveIntegration = async () => {
    try {
      const url = isEditing
        ? `/cloud-integrations/${editingId}`
        : "/cloud-integrations";

      const method = isEditing ? "PUT" : "POST";

      const res = await apiFetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (res.ok) {
        setShowFormModal(false);
        fetchData();
      } else {
        const err = await res.json();
        alert(err.error || "Failed to save integration");
      }
    } catch (e) {
      console.error(e);
      alert("An error occurred");
    }
  };

  const handleDeleteIntegration = async (id: string) => {
    if (!confirm("Are you sure? This will soft-delete the integration."))
      return;
    try {
      const res = await apiFetch(`/cloud-integrations/${id}`, {
        method: "DELETE",
      });
      if (res.ok) fetchData();
    } catch (e) {
      console.error(e);
    }
  };

  const handleTestConnection = async (id: string) => {
    try {
      const res = await apiFetch(`/cloud-integrations/${id}/test`, {
        method: "POST",
      });
      if (res.ok) {
        alert("Success!");
        fetchData();
      } else {
        alert("Connection Failed");
      }
    } catch (e) {
      console.error(e);
    }
  };

  const openApproveModal = async (req: CloudRequest) => {
    setSelectedRequest(req);
    setOverrideGroupId(req.target_group_id);
    setOverrideGroupName(req.target_group_name);
    setShowApproveModal(true);

    const intId = req.integration_id || req.integration?.id;
    if (intId) {
      try {
        setLoadingGroups(true);
        const res = await apiFetch(`/cloud-integrations/${intId}/groups`);
        if (res.ok) {
          const groups = await res.json();
          setAvailableGroups(groups || []);
        }
      } catch (e) {
        console.error(e);
      } finally {
        setLoadingGroups(false);
      }
    }
  };

  const handleConfirmApproval = async () => {
    if (!selectedRequest) return;
    try {
      const res = await apiFetch(`/cloud-requests/${selectedRequest.id}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          target_group_id: overrideGroupId,
          target_group_name: overrideGroupName,
        }),
      });
      if (res.ok) {
        setShowApproveModal(false);
        fetchData();
      } else {
        const err = await res.json();
        alert(err.error || "Failed to approve");
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleRevoke = async (id: string) => {
    if (!confirm("Revoke access immediately?")) return;
    try {
      const res = await apiFetch(`/cloud-requests/${id}/revoke`, {
        method: "POST",
      });
      if (res.ok) fetchData();
    } catch (e) {
      console.error(e);
    }
  };

  const handleReject = async (id: string) => {
    if (!confirm("Reject this pending request?")) return;
    try {
      const res = await apiFetch(`/cloud-requests/${id}`, {
        method: "DELETE",
      });
      if (res.ok) fetchData();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Cloud className="w-6 h-6" /> Cloud Integrations
          </h2>
          <p className="text-muted-foreground mt-1">
            Manage AWS, GCP, and Azure connections.
          </p>
        </div>
        <div className="flex items-center gap-4">
          <Button
            onClick={fetchData}
            variant="outline"
            size="icon"
            disabled={loading}
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
          <Button onClick={openAddModal} className="gap-2">
            <Plus className="w-4 h-4" /> Add Integration
          </Button>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {integrations.map((int) => (
          <Card
            key={int.id}
            className="bg-card/60 backdrop-blur-sm border-border"
          >
            <CardHeader className="pb-3">
              <div className="flex justify-between items-start">
                <div>
                  <CardTitle className="text-lg">{int.name}</CardTitle>
                  <CardDescription className="uppercase mt-1 font-semibold">
                    {int.provider}
                  </CardDescription>
                </div>
                <Badge
                  variant={int.status === "active" ? "default" : "destructive"}
                >
                  {int.status}
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2 mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  className="flex-1"
                  onClick={() => handleTestConnection(int.id)}
                >
                  Test
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => openEditModal(int)}
                >
                  <Edit2 className="w-4 h-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="text-destructive"
                  onClick={() => handleDeleteIntegration(int.id)}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="mt-12 mb-6">
        <h3 className="text-xl font-bold tracking-tight">Access Requests</h3>
      </div>

      <Card className="bg-card/60 backdrop-blur-sm border-border">
        <CardContent className="p-0 overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="text-xs text-muted-foreground uppercase bg-muted/50 border-b">
              <tr>
                <th className="px-6 py-4">User</th>
                <th className="px-6 py-4">Integration</th>
                <th className="px-6 py-4">Group</th>
                <th className="px-6 py-4">Status</th>
                <th className="px-6 py-4 text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {requests.map((req) => (
                <tr
                  key={req.id}
                  className="border-b border-border last:border-0"
                >
                  <td className="px-6 py-4">
                    <div className="font-medium">{req.user?.name}</div>
                    <div className="text-xs text-muted-foreground">
                      {req.user?.email}
                    </div>
                  </td>
                  <td className="px-6 py-4 uppercase text-xs">
                    {req.integration?.provider} - {req.integration?.name}
                  </td>
                  <td className="px-6 py-4 font-mono text-xs">
                    {req.target_group_name}
                  </td>
                  <td className="px-6 py-4">
                    <Badge variant="outline">{req.status}</Badge>
                  </td>
                  <td className="px-6 py-4 text-right">
                    {req.status === "pending" && (
                      <div className="flex gap-2 justify-end">
                        <Button
                          variant="outline"
                          size="sm"
                          className="text-emerald-500"
                          onClick={() => openApproveModal(req)}
                        >
                          Approve
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          className="text-destructive"
                          onClick={() => handleReject(req.id)}
                        >
                          Reject
                        </Button>
                      </div>
                    )}
                    {req.status === "active" && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="text-destructive"
                        onClick={() => handleRevoke(req.id)}
                      >
                        Revoke
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>

      {/* Add/Edit Modal */}
      {showFormModal && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <Card className="w-full max-w-lg shadow-2xl">
            <CardHeader>
              <CardTitle>{isEditing ? "Edit" : "Add"} Integration</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <input
                className="w-full bg-background border rounded p-2 text-sm"
                placeholder="Name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
              />
              {!isEditing && (
                <select
                  className="w-full bg-background border rounded p-2 text-sm"
                  value={formData.provider}
                  onChange={(e) =>
                    setFormData({ ...formData, provider: e.target.value })
                  }
                >
                  <option value="aws">AWS Identity Center</option>
                  <option value="aws-iam">AWS IAM (Legacy)</option>
                  <option value="azure">Azure Entra ID</option>
                  <option value="gcp">Google Cloud</option>
                </select>
              )}
              <textarea
                className="w-full bg-background border rounded p-2 text-sm font-mono min-h-[100px]"
                placeholder={
                  isEditing
                    ? "Credentials (JSON) - Leave blank to keep existing"
                    : "Credentials (JSON)"
                }
                value={formData.credentials}
                onChange={(e) =>
                  setFormData({ ...formData, credentials: e.target.value })
                }
              />
              <textarea
                className="w-full bg-background border rounded p-2 text-sm font-mono min-h-[80px]"
                placeholder="Metadata (JSON)"
                value={formData.metadata}
                onChange={(e) =>
                  setFormData({ ...formData, metadata: e.target.value })
                }
              />
              <div className="flex justify-end gap-2 pt-4">
                <Button variant="ghost" onClick={() => setShowFormModal(false)}>
                  Cancel
                </Button>
                <Button onClick={handleSaveIntegration}>Save</Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Approval Modal */}
      {showApproveModal && selectedRequest && (
        <div className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <Card className="w-full max-w-lg shadow-2xl">
            <CardHeader>
              <CardTitle>Approve Access</CardTitle>
              <CardDescription>
                Optionally change the group before approving.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="p-3 bg-muted rounded text-xs space-y-1">
                <p>User: {selectedRequest.user?.email}</p>
                <p>Reason: {selectedRequest.reason}</p>
              </div>

              {loadingGroups ? (
                <div className="p-4 text-center animate-pulse">
                  Loading Groups...
                </div>
              ) : availableGroups.length > 0 ? (
                <select
                  className="w-full bg-background border rounded p-2 text-sm"
                  value={overrideGroupId}
                  onChange={(e) => {
                    setOverrideGroupId(e.target.value);
                    const g = availableGroups.find(
                      (x) => x.id === e.target.value,
                    );
                    if (g) setOverrideGroupName(g.name);
                  }}
                >
                  {availableGroups.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.name}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  className="w-full bg-background border rounded p-2 text-sm"
                  value={overrideGroupId}
                  onChange={(e) => setOverrideGroupId(e.target.value)}
                  placeholder="Target Group ID"
                />
              )}

              <div className="flex justify-end gap-2 pt-4">
                <Button
                  variant="ghost"
                  onClick={() => setShowApproveModal(false)}
                >
                  Cancel
                </Button>
                <Button
                  className="bg-emerald-600 hover:bg-emerald-700 text-white"
                  onClick={handleConfirmApproval}
                >
                  Confirm Approval
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </>
  );
}
