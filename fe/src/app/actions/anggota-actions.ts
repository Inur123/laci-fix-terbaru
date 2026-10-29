"use server";

import { revalidatePath } from "next/cache";
import { backendDataOrNull, backendRequest, errorMessage, queryString } from "@/lib/backend";

export async function getAnggotaList(query?: string, page = 1, limit = 10, userId?: string, periodeId?: string, sortKey?: string | null, sortDir?: "asc" | "desc", includeRelations = false) {
  return backendRequest<any>(`/api/anggota${queryString({ q: query, page, limit, userId, periodeId, sortKey, sortDir, includeRelations: includeRelations || undefined })}`);
}

export async function getActiveUsers() {
  return (await backendRequest<any>("/api/anggota/active-users")).data ?? [];
}

export async function getAnggotaById(id: string) {
  return backendDataOrNull(`/api/anggota/${encodeURIComponent(id)}`);
}

async function save(path: string, method: "POST" | "PATCH", formData: FormData) {
  try {
    const result = await backendRequest<any>(path, { method, body: formData });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan data anggota.") };
  }
}

export async function createAnggota(formData: FormData) {
  return save("/api/anggota", "POST", formData);
}

export async function updateAnggota(id: string, formData: FormData) {
  return save(`/api/anggota/${encodeURIComponent(id)}`, "PATCH", formData);
}

export async function deleteAnggota(id: string) {
  try {
    const result = await backendRequest<any>(`/api/anggota/${encodeURIComponent(id)}`, { method: "DELETE" });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus data anggota.") };
  }
}

export async function getAnggotaStats(userId?: string) {
  return backendRequest<any>(`/api/anggota/stats${queryString({ userId })}`);
}

export async function copyAnggotaToCurrentPeriode(anggotaIds: string[]) {
  try {
    const result = await backendRequest<any>("/api/anggota/copy", { method: "POST", body: { ids: anggotaIds } });
    revalidatePath("/dashboard/anggota");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyalin data anggota.") };
  }
}
