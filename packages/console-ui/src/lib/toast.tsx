"use client";

import { toast, type ExternalToast } from "sonner";
import { CheckCircle, AlertCircle, Loader2, Info, XCircle } from "lucide-react";

export type ToastType = "success" | "error" | "warning" | "info" | "loading";

export interface MutationToastOptions extends Omit<ExternalToast, "description"> {
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  promise?: Promise<unknown>;
  onSuccess?: (data: unknown) => void;
  onError?: (error: Error) => void;
}

const defaultOptions: ExternalToast = {
  duration: 5000,
  dismissible: true,
  closeButton: true,
};

export function showSuccess(message: string, options?: MutationToastOptions) {
  return toast.success(message, {
    ...defaultOptions,
    ...options,
    icon: options?.icon ?? <CheckCircle className="size-4" />,
  });
}

export function showError(message: string, options?: MutationToastOptions) {
  return toast.error(message, {
    ...defaultOptions,
    ...options,
    icon: options?.icon ?? <AlertCircle className="size-4" />,
  });
}

export function showWarning(message: string, options?: MutationToastOptions) {
  return toast.warning(message, {
    ...defaultOptions,
    ...options,
    icon: options?.icon ?? <AlertCircle className="size-4" />,
  });
}

export function showInfo(message: string, options?: MutationToastOptions) {
  return toast.info(message, {
    ...defaultOptions,
    ...options,
    icon: options?.icon ?? <Info className="size-4" />,
  });
}

export function showLoading(message: string, options?: MutationToastOptions) {
  return toast.loading(message, {
    ...defaultOptions,
    ...options,
    icon: <Loader2 className="size-4 animate-spin" />,
  });
}

export function showPromise<T>(
  promise: Promise<T>,
  messages: {
    loading: string;
    success: string | ((data: T) => string);
    error: string | ((error: Error) => string);
  },
  options?: Omit<MutationToastOptions, "promise" | "onSuccess" | "onError">
) {
  return toast.promise(promise, {
    loading: messages.loading,
    success: messages.success,
    error: messages.error,
    ...defaultOptions,
    ...options,
  });
}

export function dismissToast(toastId: string | number) {
  toast.dismiss(toastId);
}

export function dismissAllToasts() {
  toast.dismiss();
}

export interface ApiErrorLike {
  message: string;
  status?: number;
  code?: string;
  details?: unknown;
}

function extractErrorMessage(error: unknown): string {
  if (typeof error === "string") return error;
  if (error instanceof Error) return error.message;
  if (error && typeof error === "object" && "message" in error) {
    return String((error as Record<string, unknown>).message);
  }
  return "An unexpected error occurred";
}

function extractErrorDetails(error: unknown): string | undefined {
  if (error && typeof error === "object" && "details" in error) {
    const details = (error as Record<string, unknown>).details;
    if (details) return JSON.stringify(details);
  }
  return undefined;
}

export function handleMutationError(
  error: unknown,
  fallbackMessage = "Operation failed",
  options?: MutationToastOptions
) {
  const message = extractErrorMessage(error);
  const details = extractErrorDetails(error);

  return showError(message || fallbackMessage, {
    ...options,
    description: details,
  });
}

export function handleMutationSuccess(
  message: string,
  options?: MutationToastOptions
) {
  return showSuccess(message, options);
}

export async function withToast<T>(
  promise: Promise<T>,
  options: {
    loading?: string;
    success?: string | ((data: T) => string);
    error?: string | ((error: Error) => string);
    onSuccess?: (data: T) => void;
    onError?: (error: Error) => void;
  } = {}
) {
  const { loading = "Processing...", success, error, onSuccess, onError } = options;

  const toastId = showLoading(loading);

  try {
    const data = await promise;

    toast.dismiss(toastId);

    if (success) {
      const successMessage = typeof success === "function" ? success(data) : success;
      showSuccess(successMessage);
    }

    onSuccess?.(data);
    return data;
  } catch (err) {
    toast.dismiss(toastId);

    const errorMessage = err instanceof Error ? err : new Error(extractErrorMessage(err));
    const errorMsg = error
      ? typeof error === "function"
        ? error(errorMessage)
        : error
      : "Operation failed";

    showError(errorMsg);

    onError?.(errorMessage);
    throw err;
  }
}

export function useMutationToast() {
  return {
    success: showSuccess,
    error: showError,
    warning: showWarning,
    info: showInfo,
    loading: showLoading,
    promise: showPromise,
    handleError: handleMutationError,
    handleSuccess: handleMutationSuccess,
    withToast,
    dismiss: dismissToast,
    dismissAll: dismissAllToasts,
  };
}

export const mutationToast = {
  success: showSuccess,
  error: showError,
  warning: showWarning,
  info: showInfo,
  loading: showLoading,
  promise: showPromise,
  handleError: handleMutationError,
  handleSuccess: handleMutationSuccess,
  withToast,
  dismiss: dismissToast,
  dismissAll: dismissAllToasts,
};