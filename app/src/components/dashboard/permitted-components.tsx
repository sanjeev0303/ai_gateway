"use client";

import * as React from "react";
import { Lock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Permitted, usePermission, usePermissions } from "@/hooks/use-permissions";
import type { Permission } from "@/lib/permissions";

interface PermittedButtonProps {
  permission?: Permission;
  permissions?: Permission[];
  requireAll?: boolean;
  children: React.ReactNode;
  showLockWhenDisabled?: boolean;
  disabledTooltip?: string;
  className?: string;
  variant?: "default" | "outline" | "secondary" | "ghost" | "destructive" | "link";
  size?: "default" | "xs" | "sm" | "lg" | "icon" | "icon-xs" | "icon-sm" | "icon-lg";
  asChild?: boolean;
}

export function PermittedButton({
  permission,
  permissions = [],
  requireAll = false,
  children,
  showLockWhenDisabled = true,
  disabledTooltip,
  className,
  ...props
}: PermittedButtonProps) {
  const { can, canAny, canAll, isLoading, isAuthenticated } = usePermissions();

  const allPermissions = permission ? [permission, ...permissions] : permissions;
  const allowed = allPermissions.length > 0
    ? (requireAll ? canAll(allPermissions) : canAny(allPermissions))
    : true;

  const isDisabled = !allowed || isLoading || !isAuthenticated;

  return (
    <Permitted permission={permission} permissions={permissions} requireAll={requireAll} fallback={null}>
      <Button
        disabled={isDisabled}
        className={className}
        aria-disabled={isDisabled}
        title={isDisabled && disabledTooltip ? disabledTooltip : undefined}
        {...props}
      >
        {children}
        {isDisabled && showLockWhenDisabled && (
          <Lock className="ml-2 h-4 w-4 opacity-50" aria-hidden="true" />
        )}
      </Button>
    </Permitted>
  );
}

interface PermittedLinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  permission?: Permission;
  permissions?: Permission[];
  requireAll?: boolean;
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

export function PermittedLink({
  permission,
  permissions = [],
  requireAll = false,
  children,
  fallback = null,
  className,
  ...props
}: PermittedLinkProps) {
  const { can, canAny, canAll, isLoading, isAuthenticated } = usePermissions();

  const allPermissions = permission ? [permission, ...permissions] : permissions;
  const allowed = allPermissions.length > 0
    ? (requireAll ? canAll(allPermissions) : canAny(allPermissions))
    : true;

  if (!allowed || !isAuthenticated) {
    return <>{fallback}</>;
  }

  return (
    <a className={className} {...props}>
      {children}
    </a>
  );
}

interface PermittedActionProps {
  permission?: Permission;
  permissions?: Permission[];
  requireAll?: boolean;
  onAction: () => void | Promise<void>;
  children: React.ReactNode;
  fallback?: React.ReactNode;
  className?: string;
  disabledClassName?: string;
  loadingText?: string;
}

export function PermittedAction({
  permission,
  permissions = [],
  requireAll = false,
  onAction,
  children,
  fallback = null,
  className,
  disabledClassName = "opacity-50 cursor-not-allowed",
  loadingText = "Loading...",
}: PermittedActionProps) {
  const { can, canAny, canAll, isLoading: permLoading, isAuthenticated } = usePermissions();
  const [actionLoading, setActionLoading] = React.useState(false);

  const allPermissions = permission ? [permission, ...permissions] : permissions;
  const allowed = allPermissions.length > 0
    ? (requireAll ? canAll(allPermissions) : canAny(allPermissions))
    : true;

  const isDisabled = !allowed || !isAuthenticated || actionLoading || permLoading;

  const handleClick = async () => {
    if (isDisabled) return;
    setActionLoading(true);
    try {
      await onAction();
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <Permitted permission={permission} permissions={permissions} requireAll={requireAll} fallback={fallback}>
      <button
        onClick={handleClick}
        disabled={isDisabled}
        className={`${isDisabled ? disabledClassName : ""} ${className || ""}`}
        aria-disabled={isDisabled}
      >
        {actionLoading ? loadingText : children}
      </button>
    </Permitted>
  );
}

interface PermittedDropdownItemProps extends React.ComponentProps<"button"> {
  permission?: Permission;
  permissions?: Permission[];
  requireAll?: boolean;
  children: React.ReactNode;
  fallback?: React.ReactNode;
  disabledMessage?: string;
}

export function PermittedDropdownItem({
  permission,
  permissions = [],
  requireAll = false,
  children,
  fallback = null,
  disabledMessage,
  className,
  onClick,
  ...props
}: PermittedDropdownItemProps) {
  const { can, canAny, canAll, isLoading, isAuthenticated } = usePermissions();

  const allPermissions = permission ? [permission, ...permissions] : permissions;
  const allowed = allPermissions.length > 0
    ? (requireAll ? canAll(allPermissions) : canAny(allPermissions))
    : true;

  const isDisabled = !allowed || !isAuthenticated || isLoading;

  if (!allowed && !isLoading && isAuthenticated) {
    return fallback || (
      <button
        className={`${className || ""} opacity-50 cursor-not-allowed`}
        disabled
        title={disabledMessage || "You don't have permission to perform this action"}
        {...props}
      >
        {children}
        <Lock className="ml-2 h-4 w-4 opacity-50" aria-hidden="true" />
      </button>
    );
  }

  return (
    <button
      onClick={onClick}
      disabled={isDisabled}
      className={className}
      aria-disabled={isDisabled}
      title={isDisabled && disabledMessage ? disabledMessage : undefined}
      {...props}
    >
      {children}
    </button>
  );
}

export function usePermittedAction(permission?: Permission, permissions: Permission[] = [], requireAll = false) {
  const { can, canAny, canAll, isLoading, isAuthenticated } = usePermissions();

  const allPermissions = permission ? [permission, ...permissions] : permissions;
  const allowed = allPermissions.length > 0
    ? (requireAll ? canAll(allPermissions) : canAny(allPermissions))
    : true;

  return {
    allowed: allowed && isAuthenticated && !isLoading,
    isLoading: isLoading || !isAuthenticated,
  };
}