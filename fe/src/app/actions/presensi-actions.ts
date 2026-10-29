"use server";

import { revalidatePath } from "next/cache";
import { BackendError, backendDataOrNull, backendRequest, errorMessage, queryString } from "@/lib/backend";
import { isPresensiOpen as checkIsPresensiOpen } from "@/lib/presensi-utils";

export async function getPresensiList(query?: string, page = 1, limit = 10, statusFilter?: string, sortKey?: string | null, sortDir?: "asc" | "desc") {
  return backendRequest<any>(`/api/presensi${queryString({ q: query, page, limit, status: statusFilter, sortKey, sortDir })}`);
}

export async function getPresensiDetail(id: string) {
  return backendDataOrNull(`/api/presensi/${encodeURIComponent(id)}`);
}

export async function getPublicPresensiDetail(id: string) {
  return backendDataOrNull(`/api/presensi/${encodeURIComponent(id)}/public`);
}

async function save(path: string, method: "POST" | "PATCH", formData: FormData) {
  try {
    const result = await backendRequest<any>(path, { method, body: formData });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan kegiatan presensi.") };
  }
}

export async function createPresensi(formData: FormData) {
  return save("/api/presensi", "POST", formData);
}

export async function isPresensiOpen(presensi: any) {
  return checkIsPresensiOpen(presensi);
}

export async function updatePresensi(id: string, formData: FormData) {
  return save(`/api/presensi/${encodeURIComponent(id)}`, "PATCH", formData);
}

export async function submitPresensiData(presensiId: string, formData: FormData) {
  try {
    return await backendRequest<any>(`/api/presensi/${encodeURIComponent(presensiId)}/absensi`, { method: "POST", body: formData });
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan data presensi.") };
  }
}

export async function deletePresensi(id: string) {
  try {
    const result = await backendRequest<any>(`/api/presensi/${encodeURIComponent(id)}`, { method: "DELETE" });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus kegiatan presensi.") };
  }
}

export async function updatePresensiStatus(id: string, mode: "AUTO" | "MANUAL_CLOSE") {
  try {
    const result = await backendRequest<any>(`/api/presensi/${encodeURIComponent(id)}/status`, { method: "PATCH", body: { mode } });
    revalidatePath("/dashboard/presensi");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal mengubah status presensi.") };
  }
}

export async function getParticipantDetail(id: string, token?: string) {
  try {
    const suffix = token ? `?token=${encodeURIComponent(token)}` : "";
    return (await backendRequest<any>(`/api/participants/${encodeURIComponent(id)}${suffix}`)).data ?? null;
  } catch (error) {
    if (error instanceof BackendError && (error.status === 403 || error.status === 404)) return null;
    throw error;
  }
}
