import { backendRequest } from "@/lib/backend";
import { cache } from "react";

export const getSession = cache(async () => {
  try {
    return await backendRequest<any>("/api/auth/session");
  } catch {
    return null;
  }
});

export async function getCurrentUser() {
  const session = await getSession();
  return session?.user ?? null;
}

export async function requireAuth() {
  const session = await getSession();
  if (!session?.user) {
    throw new Error("Unauthorized");
  }

  if ((session.user as any).isActive === false) {
    throw new Error("Inactive account");
  }

  return session;
}

export async function requireRole(role: string) {
  const session = await requireAuth();
  if ((session.user as any).role !== role) {
    throw new Error("Forbidden");
  }
  return session;
}
