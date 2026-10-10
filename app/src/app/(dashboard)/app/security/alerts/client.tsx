"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { createAlert, updateAlert, deleteAlert, testAlert } from "./alert-actions";
import { AlertSeverity, AlertStatus, AlertType } from "@ai-governance/api-client";
import { Textarea } from "@/components/ui/textarea";
import { useRouter } from "next/navigation";

export function CreateAlertDialog() {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [alertType, setAlertType] = useState<AlertType>("spend_threshold");
  const [severity, setSeverity] = useState<AlertSeverity>("warning");
  const [config, setConfig] = useState("{}");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const parsedConfig = JSON.parse(config);
      await createAlert({ name, alert_type: alertType, severity, configuration: parsedConfig });
      setOpen(false);
      setName("");
      setConfig("{}");
    } catch (err: any) {
      setError(err.message || "Failed to create alert");
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Button onClick={() => setOpen(true)}>Create alert</Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>Create Alert Rule</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              {error && <div className="text-sm text-destructive">{error}</div>}
              <div className="grid gap-2">
                <Label htmlFor="name">Name</Label>
                <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="alertType">Alert Type</Label>
                <Select value={alertType} onValueChange={(v) => setAlertType(v as AlertType)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="spend_threshold">Spend threshold</SelectItem>
                    <SelectItem value="spend_exceeded">Spend exceeded</SelectItem>
                    <SelectItem value="request_rate_spike">Request rate spike</SelectItem>
                    <SelectItem value="error_rate_spike">Error rate spike</SelectItem>
                    <SelectItem value="provider_failure">Provider failure</SelectItem>
                    <SelectItem value="credential_failure">Credential failure</SelectItem>
                    <SelectItem value="policy_violation">Policy violation</SelectItem>
                    <SelectItem value="rate_limit_spike">Rate limit spike</SelectItem>
                    <SelectItem value="usage_anomaly">Usage anomaly</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="severity">Severity</Label>
                <Select value={severity} onValueChange={(v) => setSeverity(v as AlertSeverity)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warning">Warning</SelectItem>
                    <SelectItem value="critical">Critical</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="config">Configuration (JSON)</Label>
                <Textarea id="config" value={config} onChange={(e) => setConfig(e.target.value)} required />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={loading}>{loading ? "Saving..." : "Create"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function AlertActionsMenu({ alert }: { alert: any }) {
  const [editOpen, setEditOpen] = useState(false);
  const [testResult, setTestResult] = useState<any>(null);
  
  const [name, setName] = useState(alert.name);
  const [severity, setSeverity] = useState<AlertSeverity>(alert.severity);
  const [status, setStatus] = useState<AlertStatus>(alert.status);
  const [config, setConfig] = useState(alert.configuration ? JSON.stringify(alert.configuration, null, 2) : "{}");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleEditSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const parsedConfig = JSON.parse(config);
      await updateAlert(alert.id, { name, severity, status, configuration: parsedConfig });
      setEditOpen(false);
    } catch (err: any) {
      setError(err.message || "Failed to update alert");
    } finally {
      setLoading(false);
    }
  };

  const toggleStatus = async () => {
    const newStatus = alert.status === "active" ? "muted" : "active";
    await updateAlert(alert.id, { 
      name: alert.name, 
      severity: alert.severity, 
      status: newStatus, 
      configuration: alert.configuration 
    });
  };

  const handleTest = async () => {
    try {
      const result = await testAlert(alert.id);
      setTestResult(result);
    } catch (err: any) {
      alert("Test failed: " + err.message);
    }
  };

  return (
    <div className="flex items-center gap-2">
      <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>Edit</Button>
      <Button 
        variant="outline" 
        size="sm" 
        onClick={async () => {
          setLoading(true);
          try {
            await toggleStatus();
          } finally {
            setLoading(false);
          }
        }}
        disabled={loading}
      >
        {alert.status === "active" ? "Mute" : "Unmute"}
      </Button>
      <Button 
        variant="outline" 
        size="sm" 
        onClick={async () => {
          setLoading(true);
          try {
            await handleTest();
          } finally {
            setLoading(false);
          }
        }}
        disabled={loading}
      >
        Test
      </Button>
      <Button 
        variant="destructive" 
        size="sm" 
        onClick={async () => {
          if (confirm(`Delete alert ${alert.name}?`)) {
            setLoading(true);
            try {
              await deleteAlert(alert.id);
            } finally {
              setLoading(false);
            }
          }
        }}
        disabled={loading}
      >
        Delete
      </Button>

      {/* Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <form onSubmit={handleEditSubmit}>
            <DialogHeader>
              <DialogTitle>Edit Alert Rule</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              {error && <div className="text-sm text-destructive">{error}</div>}
              <div className="grid gap-2">
                <Label htmlFor="name">Name</Label>
                <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="severity">Severity</Label>
                <Select value={severity} onValueChange={(v) => setSeverity(v as AlertSeverity)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warning">Warning</SelectItem>
                    <SelectItem value="critical">Critical</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="status">Status</Label>
                <Select value={status} onValueChange={(v) => setStatus(v as AlertStatus)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="muted">Muted</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="config">Configuration (JSON)</Label>
                <Textarea id="config" value={config} onChange={(e) => setConfig(e.target.value)} required />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setEditOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={loading}>{loading ? "Saving..." : "Save changes"}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Test Result Dialog */}
      <Dialog open={!!testResult} onOpenChange={(v) => { if (!v) setTestResult(null) }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Alert Test Result</DialogTitle>
          </DialogHeader>
          <div className="py-4 space-y-4">
            <div className="text-sm">
              <span className="font-semibold">Message:</span> {testResult?.message}
            </div>
            <div className="text-sm">
              <span className="font-semibold">Notification Sent:</span> {testResult?.notification_sent ? "Yes" : "No"}
            </div>
            <div className="text-sm">
              <span className="font-semibold">Status Muted:</span> {testResult?.status_is_muted ? "Yes" : "No"}
            </div>
            <div className="text-sm">
              <span className="font-semibold">Config evaluated:</span>
              <pre className="mt-2 bg-muted p-2 rounded-md">{JSON.stringify(testResult?.configuration, null, 2)}</pre>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setTestResult(null)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
