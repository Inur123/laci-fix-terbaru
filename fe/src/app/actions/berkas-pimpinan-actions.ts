"use server";

import { revalidatePath } from "next/cache";
import { backendDataOrNull, backendRequest, errorMessage, queryString } from "@/lib/backend";

export async function getBerkasPimpinans(query?: string, page = 1, limit = 10, sortKey?: string | null, sortDir?: "asc" | "desc") {
  return backendRequest<any>(`/api/berkas-pimpinan${queryString({ q: query, page, limit, sortKey, sortDir })}`);
}

export async function getBerkasPimpinanById(id: string) {
  return backendDataOrNull(`/api/berkas-pimpinan/${encodeURIComponent(id)}`);
}

async function save(path: string, method: "POST" | "PATCH", formData: FormData) {
  try {
    const result = await backendRequest<any>(path, { method, body: formData });
    revalidatePath("/dashboard/berkas-pimpinan");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan berkas pimpinan.") };
  }
}

export async function createBerkasPimpinan(formData: FormData) {
  return save("/api/berkas-pimpinan", "POST", formData);
}

export async function updateBerkasPimpinan(id: string, formData: FormData) {
  return save(`/api/berkas-pimpinan/${encodeURIComponent(id)}`, "PATCH", formData);
}

export async function deleteBerkasPimpinan(id: string) {
  try {
    const result = await backendRequest<any>(`/api/berkas-pimpinan/${encodeURIComponent(id)}`, { method: "DELETE" });
    revalidatePath("/dashboard/berkas-pimpinan");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus berkas pimpinan.") };
  }
}

export async function getBerkasPimpinanDownloadToken(id: string) {
  return (await backendRequest<any>(`/api/berkas-pimpinan/${encodeURIComponent(id)}/download-token`, { method: "POST" })).token as string;
}
