"use server";

import { revalidatePath } from "next/cache";
import {
  backendDataOrNull,
  backendRequest,
  errorMessage,
  queryString,
} from "@/lib/backend";

export type KegiatanItem = {
  id: string;
  judul: string;
  deskripsi?: string | null;
  lokasi?: string | null;
  warna: string;
  tanggalMulai: Date | string;
  tanggalSelesai?: Date | string | null;
};

export type KegiatanListItem = KegiatanItem & { status: string };

export type KegiatanPage = {
  data: KegiatanListItem[];
  total: number;
  totalPages: number;
};

export type KegiatanStats = {
  total: number;
  mendatang: number;
  selesai: number;
};

type MutationResult = {
  success?: string;
  error?: string;
  data?: { id: string };
};

export async function getAgendaKegiatanList(
  query?: string,
  page = 1,
  limit = 10,
  statusFilter?: string,
  sortKey?: string | null,
  sortDir?: "asc" | "desc",
): Promise<KegiatanPage> {
  return backendRequest<KegiatanPage>(
    `/api/agenda-kegiatan${queryString({
      q: query,
      page,
      limit,
      status: statusFilter,
      sortKey,
      sortDir,
    })}`,
  );
}

export async function getKegiatanById(id: string) {
  return backendDataOrNull<KegiatanItem>(
    `/api/agenda-kegiatan/${encodeURIComponent(id)}`,
  );
}

async function saveKegiatan(
  path: string,
  method: "POST" | "PATCH",
  formData: FormData,
): Promise<MutationResult> {
  try {
    const result = await backendRequest<MutationResult>(path, {
      method,
      body: formData,
    });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan kegiatan.") };
  }
}

export async function createKegiatan(formData: FormData) {
  return saveKegiatan("/api/agenda-kegiatan", "POST", formData);
}

export async function updateKegiatan(id: string, formData: FormData) {
  return saveKegiatan(
    `/api/agenda-kegiatan/${encodeURIComponent(id)}`,
    "PATCH",
    formData,
  );
}

export async function deleteKegiatan(id: string) {
  try {
    const result = await backendRequest<MutationResult>(
      `/api/agenda-kegiatan/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    );
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus kegiatan.") };
  }
}

export async function getAgendaKegiatanStats() {
  return backendRequest<KegiatanStats>("/api/agenda-kegiatan/stats");
}
