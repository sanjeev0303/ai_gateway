"use client";

import * as React from "react";
import {
  AlertTriangle,
  Bell,
  CheckCircle,
  Clock,
  ShieldAlert,
  Check,
  CheckCheck,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { toast } from "sonner";
import {
  acknowledgeAlertEventAction,
  resolveAlertEventAction,
  markNotificationReadAction,
  markAllNotificationsReadAction,
} from "./alert-actions";
import type { AlertEvent, InAppNotification } from "@ai-governance/api-client";
import { formatRelative } from "@/lib/format";

interface IncidentsClientProps {
  initialEvents: AlertEvent[];
  initialNotifications: InAppNotification[];
}

export function IncidentsClient({
  initialEvents,
  initialNotifications,
}: IncidentsClientProps) {
  const [events, setEvents] = React.useState<AlertEvent[]>(initialEvents);
  const [notifications, setNotifications] =
    React.useState<InAppNotification[]>(initialNotifications);
  const [loadingId, setLoadingId] = React.useState<string | null>(null);

  const handleAcknowledge = async (id: string) => {
    setLoadingId(id);
    try {
      const res = await acknowledgeAlertEventAction(id);
      toast.success("Incident acknowledged");
      setEvents((prev) => prev.map((e) => (e.id === id ? res : e)));
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to acknowledge");
    } finally {
      setLoadingId(null);
    }
  };

  const handleResolve = async (id: string) => {
    setLoadingId(id);
    try {
      const res = await resolveAlertEventAction(id);
      toast.success("Incident resolved");
      setEvents((prev) => prev.map((e) => (e.id === id ? res : e)));
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to resolve");
    } finally {
      setLoadingId(null);
    }
  };

  const handleMarkNotificationRead = async (id: string) => {
    try {
      await markNotificationReadAction(id);
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, read: true } : n))
      );
      toast.success("Notification marked as read");
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to mark read");
    }
  };

  const handleMarkAllRead = async () => {
    try {
      await markAllNotificationsReadAction();
      setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
      toast.success("All notifications marked as read");
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed to mark all read");
    }
  };

  return (
    <div className="space-y-6">
      {/* Incidents Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <ShieldAlert className="h-5 w-5 text-destructive" />
                Active Incidents & Triggered Events
              </CardTitle>
              <CardDescription>
                Real-time incident states dispatched by the Sentinel Watchdog evaluator.
              </CardDescription>
            </div>
            <Badge variant="outline" className="font-mono">
              {events.filter((e) => e.status !== "resolved").length} Unresolved
            </Badge>
          </div>
        </CardHeader>
        <CardContent>
          {events.length === 0 ? (
            <div className="text-center py-8 space-y-2">
              <CheckCircle className="h-8 w-8 text-green-500 mx-auto" />
              <p className="text-sm font-medium">All systems normal</p>
              <p className="text-xs text-muted-foreground">
                No active watchdog incidents or anomaly conditions have triggered.
              </p>
            </div>
          ) : (
            <div className="divide-y rounded-md border">
              {events.map((e) => (
                <div
                  key={e.id}
                  className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4 hover:bg-muted/30 transition-colors"
                >
                  <div className="space-y-1.5 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge
                        variant={
                          e.severity === "critical"
                            ? "destructive"
                            : e.severity === "warning"
                            ? "default"
                            : "outline"
                        }
                        className="text-[11px] uppercase"
                      >
                        {e.severity}
                      </Badge>
                      <Badge
                        variant={
                          e.status === "triggered"
                            ? "destructive"
                            : e.status === "acknowledged"
                            ? "outline"
                            : "secondary"
                        }
                        className="text-[11px] uppercase font-mono"
                      >
                        {e.status}
                      </Badge>
                      <span className="font-medium text-sm">{e.message}</span>
                    </div>

                    <div className="text-xs text-muted-foreground flex flex-wrap items-center gap-3">
                      <span>Type: <code className="text-foreground">{e.alert_type}</code></span>
                      <span>Triggered: {formatRelative(e.triggered_at, new Date())}</span>
                      {e.resolved_at && (
                        <span>Resolved: {formatRelative(e.resolved_at, new Date())}</span>
                      )}
                    </div>

                    {e.context && Object.keys(e.context).length > 0 && (
                      <div className="text-[11px] bg-muted/50 p-2 rounded max-w-xl font-mono text-muted-foreground overflow-x-auto">
                        {JSON.stringify(e.context)}
                      </div>
                    )}
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    {e.status === "triggered" && (
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={loadingId === e.id}
                        onClick={() => handleAcknowledge(e.id)}
                      >
                        <Clock className="h-3.5 w-3.5 mr-1" />
                        Acknowledge
                      </Button>
                    )}
                    {e.status !== "resolved" && (
                      <Button
                        size="sm"
                        disabled={loadingId === e.id}
                        onClick={() => handleResolve(e.id)}
                      >
                        <Check className="h-3.5 w-3.5 mr-1" />
                        Resolve
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Notifications Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Bell className="h-5 w-5 text-blue-500" />
                In-App Notification Dispatch
              </CardTitle>
              <CardDescription>
                Auditable notification feed generated across multi-channel alert delivery.
              </CardDescription>
            </div>
            {notifications.some((n) => !n.read) && (
              <Button size="sm" variant="ghost" onClick={handleMarkAllRead}>
                <CheckCheck className="h-4 w-4 mr-1.5" />
                Mark all read
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {notifications.length === 0 ? (
            <div className="text-center py-6 text-sm text-muted-foreground">
              No in-app notifications.
            </div>
          ) : (
            <div className="space-y-2">
              {notifications.map((n) => (
                <div
                  key={n.id}
                  className={`p-3.5 rounded-lg border flex items-center justify-between gap-3 transition-colors ${
                    n.read ? "bg-background/40 opacity-70" : "bg-muted/40 border-primary/30"
                  }`}
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-sm">{n.title}</span>
                      <Badge variant="outline" className="text-[10px] uppercase">
                        {n.severity}
                      </Badge>
                      {!n.read && (
                        <span className="h-2 w-2 rounded-full bg-blue-500" />
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">{n.message}</p>
                    <span className="text-[10px] text-muted-foreground">
                      {formatRelative(n.created_at, new Date())}
                    </span>
                  </div>

                  {!n.read && (
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => handleMarkNotificationRead(n.id)}
                    >
                      Mark read
                    </Button>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
