"use server";

import { backendRequest } from "@/lib/backend";

export async function getDashboardStats() {
  return backendRequest<any>("/api/dashboard/stats");
}

export async function getPublicStats() {
  const result = await backendRequest<any>("/api/public/stats");
  return { anggotaCount: result.anggotaCount ?? 0, suratCount: result.suratCount ?? 0 };
}
