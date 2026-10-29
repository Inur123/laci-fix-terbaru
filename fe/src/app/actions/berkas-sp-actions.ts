"use server";

import { revalidatePath } from "next/cache";
import {
  backendDataOrNull,
  backendRequest,
  errorMessage,
  queryString,
} from "@/lib/backend";

export type BerkasSPFormData = {
  nama: string;
  organisasi: "IPNU" | "IPPNU" | null;
  tanggalMulai: Date;
  tanggalBerakhir: Date;
  catatan?: string;
  file?: File;
};

export async function getBerkasSPs(
  query?: string,
  organisasiFilter?: string,
  page = 1,
  limit = 10,
  sortKey?: string | null,
  sortDir?: "asc" | "desc",
) {
  return backendRequest<any>(
    `/api/berkas-sp${queryString({ q: query, organisasi: organisasiFilter, page, limit, sortKey, sortDir })}`,
  );
}

export async function getBerkasSPStats() {
  return backendRequest<any>("/api/berkas-sp/stats");
}

export async function getBerkasSPById(id: string) {
  return backendDataOrNull(`/api/berkas-sp/${encodeURIComponent(id)}`);
}

async function save(
  path: string,
  method: "POST" | "PATCH",
  formData: FormData,
) {
  try {
    const result = await backendRequest<any>(path, { method, body: formData });
    revalidatePath("/dashboard/berkas-sp");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan berkas SP.") };
  }
}

export async function createBerkasSP(formData: FormData) {
  return save("/api/berkas-sp", "POST", formData);
}

export async function updateBerkasSP(id: string, formData: FormData) {
  return save(`/api/berkas-sp/${encodeURIComponent(id)}`, "PATCH", formData);
}

export async function deleteBerkasSP(id: string) {
  try {
    const result = await backendRequest<any>(
      `/api/berkas-sp/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    );
    revalidatePath("/dashboard/berkas-sp");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus berkas SP.") };
  }
}

export async function getBerkasSPDownloadToken(id: string) {
  return (
    await backendRequest<any>(
      `/api/berkas-sp/${encodeURIComponent(id)}/download-token`,
      { method: "POST" },
    )
  ).token as string;
}
