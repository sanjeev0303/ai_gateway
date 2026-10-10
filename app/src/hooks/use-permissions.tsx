"use client";

import * as React from "react";
import { useSession } from "@/lib/session-context";
import type { Role, Permission } from "@/lib/permissions";
import {
  hasPermission as checkPermission,
  hasAnyPermission,
  hasAllPermissions,
  getPermissionsForRole,
  PERMISSION_GROUPS,
} from "@/lib/permissions";

function normalizeRole(role: string): Role {
  const validRoles: Role[] = ["owner", "admin", "auditor", "billing_admin", "member", "service"];
  return validRoles.includes(role as Role) ? (role as Role) : "member";
}

export function usePermissions() {
  const session = useSession();

  const role = React.useMemo(() => {
    if (session.status !== "authenticated") return "member" as Role;
    return normalizeRole(session.session.user.role);
  }, [session.status, session.status === "authenticated" ? session.session.user.role : "member"]);

  const permissions = React.useMemo(() => getPermissionsForRole(role), [role]);

  const can = React.useCallback(
    (permission: Permission): boolean => {
      return checkPermission(role, permission);
    },
    [role]
  );

  const canAny = React.useCallback(
    (permissions: Permission[]): boolean => {
      return hasAnyPermission(role, permissions);
    },
    [role]
  );

  const canAll = React.useCallback(
    (permissions: Permission[]): boolean => {
      return hasAllPermissions(role, permissions);
    },
    [role]
  );

  const cannot = React.useCallback(
    (permission: Permission): boolean => {
      return !checkPermission(role, permission);
    },
    [role]
  );

  return {
    role,
    permissions,
    can,
    cannot,
    canAny,
    canAll,
    isAuthenticated: session.status === "authenticated",
    isLoading: session.status === "loading",
  };
}

export function usePermission(permission: Permission) {
  const { can, isLoading, isAuthenticated } = usePermissions();
  return {
    allowed: can(permission),
    isLoading,
    isAuthenticated,
  };
}

export function usePermissionsForGroup(group: keyof typeof PERMISSION_GROUPS) {
  const { canAny, canAll, isLoading, isAuthenticated } = usePermissions();
  const permissions = PERMISSION_GROUPS[group] || [];

  return {
    canAny: canAny(permissions),
    canAll: canAll(permissions),
    permissions,
    isLoading,
    isAuthenticated,
  };
}

interface PermissionGuardProps {
  permission?: Permission;
  permissions?: Permission[];
  requireAll?: boolean;
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

export function PermissionGuard({
  permission,
  permissions = [],
  requireAll = false,
  children,
  fallback = null,
}: PermissionGuardProps) {
  const { can, canAny, canAll, isLoading, isAuthenticated } = usePermissions();

  if (isLoading) {
    return <>{children}</>;
  }

  if (!isAuthenticated) {
    return <>{fallback}</>;
  }

  const allPermissions = permission ? [permission, ...permissions] : permissions;

  if (allPermissions.length === 0) {
    return <>{children}</>;
  }

  const allowed = requireAll ? canAll(allPermissions) : canAny(allPermissions);

  return allowed ? <>{children}</> : <>{fallback}</>;
}

interface PermittedProps {
  permission?: Permission;
  permissions?: Permission[];
  requireAll?: boolean;
  children: React.ReactNode;
  fallback?: React.ReactNode;
  asChild?: boolean;
}

export function Permitted({
  permission,
  permissions = [],
  requireAll = false,
  children,
  fallback = null,
  asChild = false,
}: PermittedProps) {
  const { can, canAny, canAll, isLoading, isAuthenticated } = usePermissions();

  if (isLoading || !isAuthenticated) {
    return <>{fallback}</>;
  }

  const allPermissions = permission ? [permission, ...permissions] : permissions;

  if (allPermissions.length === 0) {
    return <>{children}</>;
  }

  const allowed = requireAll ? canAll(allPermissions) : canAny(allPermissions);

  if (!allowed) {
    return <>{fallback}</>;
  }

  if (asChild && React.isValidElement(children)) {
    return React.cloneElement(children, children.props as Record<string, unknown>);
  }

  return <>{children}</>;
}