"use server";

import { revalidatePath } from "next/cache";
import { backendDataOrNull, backendRequest, errorMessage, queryString } from "@/lib/backend";

export type ArsipSuratFormData = {
  organisasi: "IPNU" | "IPPNU" | "BERSAMA" | "CBP_KPP" | null;
  noSurat: string;
  jenisSurat: "MASUK" | "KELUAR";
  tanggal: Date;
  pengirimPenerima: string;
  deskripsi?: string;
  perihal: string;
  file?: File;
};

export async function getArsipSurats(query?: string, organisasiFilter?: string, jenisSuratFilter?: string, page = 1, limit = 10, sortKey?: string | null, sortDir?: "asc" | "desc") {
  return backendRequest<any>(`/api/arsip${queryString({ q: query, organisasi: organisasiFilter, jenis: jenisSuratFilter, page, limit, sortKey, sortDir })}`);
}

export async function getArsipStats() {
  return backendRequest<any>("/api/arsip/stats");
}

export async function getArsipSuratById(id: string) {
  return backendDataOrNull(`/api/arsip/${encodeURIComponent(id)}`);
}

async function saveArsip(path: string, method: "POST" | "PATCH", formData: FormData) {
  try {
    const result = await backendRequest<any>(path, { method, body: formData });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan arsip surat.") };
  }
}

export async function createArsipSurat(formData: FormData) {
  return saveArsip("/api/arsip", "POST", formData);
}

export async function updateArsipSurat(id: string, formData: FormData) {
  return saveArsip(`/api/arsip/${encodeURIComponent(id)}`, "PATCH", formData);
}

export async function deleteArsipSurat(id: string) {
  try {
    const result = await backendRequest<any>(`/api/arsip/${encodeURIComponent(id)}`, { method: "DELETE" });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus arsip surat.") };
  }
}

export async function getArsipDownloadToken(id: string) {
  return (await backendRequest<any>(`/api/arsip/${encodeURIComponent(id)}/download-token`, { method: "POST" })).token as string;
}
