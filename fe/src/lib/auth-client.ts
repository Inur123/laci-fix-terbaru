"use client";

import { useEffect, useState } from "react";

type AuthError = { status: number; code?: string; message: string };
type AuthResult<T = any> = { data: T | null; error: AuthError | null };

async function request<T>(path: string, init?: RequestInit): Promise<AuthResult<T>> {
  try {
    const response = await fetch(path, {
      ...init,
      credentials: "include",
      headers: { "content-type": "application/json", ...init?.headers },
    });
    const payload = await response.json().catch(() => null);
    if (!response.ok) {
      return {
        data: null,
        error: {
          status: response.status,
          code: payload?.code,
          message: payload?.message || "Permintaan gagal diproses.",
        },
      };
    }
    return { data: payload, error: null };
  } catch {
    return { data: null, error: { status: 0, code: "NETWORK_ERROR", message: "Backend tidak dapat dihubungi." } };
  }
}

export const authClient = {
  signIn: {
    email: ({ email, password }: { email: string; password: string }) =>
      request<any>("/api/auth/login", { method: "POST", body: JSON.stringify({ email, password }) }),
    social: async ({ callbackURL }: { provider: string; callbackURL?: string; errorCallbackURL?: string }) => {
      const callback = callbackURL || `${window.location.origin}/dashboard?login=success`;
      window.location.assign(`/api/auth/google?callback=${encodeURIComponent(callback)}`);
      return { data: null, error: null } as AuthResult;
    },
  },
  signUp: {
    email: ({ name, email, password }: { name: string; email: string; password: string }) =>
      request<any>("/api/auth/register", { method: "POST", body: JSON.stringify({ name, email, password }) }),
  },
  signOut: async (options?: { fetchOptions?: { onSuccess?: () => void } }) => {
    const result = await request("/api/auth/logout", { method: "POST", body: "{}" });
    if (!result.error) options?.fetchOptions?.onSuccess?.();
    return result;
  },
  getSession: () => request<any>("/api/auth/session"),
  emailOtp: {
    sendVerificationOtp: ({ email }: { email: string; type?: string }) =>
      request("/api/auth/otp/send", { method: "POST", body: JSON.stringify({ email }) }),
    verifyEmail: ({ email, otp }: { email: string; otp: string }) =>
      request("/api/auth/otp/verify", { method: "POST", body: JSON.stringify({ email, otp }) }),
  },
};

export function useSession() {
  const [data, setData] = useState<any>(null);
  const [isPending, setIsPending] = useState(true);

  useEffect(() => {
    let active = true;
    const refresh = async () => {
      const result = await authClient.getSession();
      if (active) {
        setData(result.data);
        setIsPending(false);
      }
    };
    void refresh();
    const timer = window.setInterval(refresh, 60_000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, []);

  return { data, isPending };
}

export const signIn = authClient.signIn;
export const signUp = authClient.signUp;
export const signOut = authClient.signOut;
