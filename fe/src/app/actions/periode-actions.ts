"use server";

import { revalidatePath } from "next/cache";
import { backendDataOrNull, backendRequest, errorMessage } from "@/lib/backend";

export async function createPeriode(nama: string) {
  try {
    const result = await backendRequest<any>("/api/periodes", { method: "POST", body: { nama } });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal membuat periode.") };
  }
}

export async function activatePeriode(periodeId: string) {
  try {
    const result = await backendRequest<any>(`/api/periodes/${encodeURIComponent(periodeId)}/activate`, { method: "POST" });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal mengaktifkan periode.") };
  }
}

export async function deletePeriode(periodeId: string) {
  try {
    const result = await backendRequest<any>(`/api/periodes/${encodeURIComponent(periodeId)}`, { method: "DELETE" });
    revalidatePath("/dashboard/periode");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus periode.") };
  }
}

export async function getPeriode(id: string) {
  return backendDataOrNull(`/api/periodes/${encodeURIComponent(id)}`);
}

export async function getPeriodes(page = 1, limit = 100) {
  const result = await backendRequest<any>(`/api/periodes?page=${page}&limit=${limit}`);
  return result.data ?? [];
}

export async function updatePeriode(id: string, nama: string) {
  try {
    const result = await backendRequest<any>(`/api/periodes/${encodeURIComponent(id)}`, { method: "PATCH", body: { nama } });
    revalidatePath("/dashboard/periode");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal memperbarui periode.") };
  }
}
