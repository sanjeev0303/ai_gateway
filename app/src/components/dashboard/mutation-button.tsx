"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/dashboard/status-badge";

export interface MutationButtonProps<TError = Error> {
  /** Server action to execute for the mutation. */
  action: (state: unknown, formData: FormData) => Promise<{ success?: boolean; error?: string }>;
  /** Label for the submit/primary action. */
  submitLabel?: string;
  /** Variant of the button (default, destructive, secondary, etc.). */
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
  /** Size of the button (default, sm, lg, icon, etc.). */
  size?: "default" | "xs" | "sm" | "lg" | "icon";
  /** Optional onSuccess callback after mutation succeeds. */
  onSuccess?: () => void;
  /** Optional onError callback after mutation fails. */
  onError?: (error: string) => void;
  /** Optional children to render inside the button (e.g. an icon). */
  children?: React.ReactNode;
  /** Disabled state. */
  disabled?: boolean;
  /** ARIA-label for the button. */
  ariaLabel?: string;
}

/**
 * MutationButton — A unified button for API mutations with loading/error states.
 *
 * Wraps Next.js's useActionState to provide:
 * - Loading skeleton with Loader2 spinner
 * - Error badge display
 * - Success handling with onSuccess callback
 * - Consistent styling with the UI button variant/size system
 * - Integrated with the existing form/action state pattern used throughout the dashboard
 *
 * Usage:
 *   <MutationButton
 *     action={updateOrganization}
 *     onSuccess={() => router.push("/organizations")}
 *   >
 *     Save changes
 *   </MutationButton>
 */
export function MutationButton<TError = Error>({
  action,
  submitLabel = "Submit",
  variant = "default",
  size = "default",
  onSuccess,
  onError,
  children,
  disabled = false,
  ariaLabel = "Submit",
}: MutationButtonProps<TError>) {
  const [state, formAction, pending] = React.useActionState(action, {});

  React.useEffect(() => {
    if (state !== undefined && "success" in state && state.success && !pending) {
      onSuccess?.();
    }
  }, [state, pending, onSuccess]);

  return (
    <Button
      variant={variant}
      size={size}
      disabled={pending || disabled}
      aria-label={ariaLabel}
    >
      {pending ? (
        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
      ) : state !== undefined && "error" in state && state.error ? (
        <span className="inline-flex items-center gap-2">
          <StatusBadge tone="danger">{state.error}</StatusBadge>
        </span>
      ) : children !== undefined
        ? children
        : submitLabel}
    </Button>
  );
}

/**
 * Convenience hook that provides pre-configured state for use with MutationButton.
 * Wraps a control API call as a server action.
 *
 * Example:
 *   const { mutateAsync } = useOrgMutation({
 *     onSuccess: () => router.push("/"),
 *   });
 *   <MutationButton action={mutateAsync} onSuccess={() => router.push("/")}>
 *     Save changes
 *   </MutationButton>
 */
export function useMutationButton(
  action: (state: unknown, formData: FormData) => Promise<{ success?: boolean; error?: string }>,
  options?: {
    onSuccess?: () => void;
    onError?: (error: string) => void;
  }
) {
  const [state, formAction, pending] = React.useActionState(action, {});

  React.useEffect(() => {
    if (state !== undefined && "success" in state && state.success && !pending) {
      options?.onSuccess?.();
    }
  }, [state, pending, options?.onSuccess]);

  return { state, formAction, pending };
}