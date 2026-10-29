"use server";

import { revalidatePath } from "next/cache";
import {
  backendDataOrNull,
  backendRequest,
  errorMessage,
  queryString,
} from "@/lib/backend";
import type {
  PengajuanBerkas,
  PengajuanBerkasStats,
  StatusPengajuan,
} from "@/types/domain";

type PacUser = { id: string; name: string };
type PengajuanPage = {
  data: PengajuanBerkas[];
  total: number;
  totalPages: number;
};
type MutationResult = {
  success?: string;
  error?: string;
  data?: { id: string };
};

export async function getActivePacUsers() {
  const result = await backendRequest<{ data: PacUser[] }>(
    "/api/pengajuan/active-pac",
  );
  return result.data ?? [];
}

export async function getActivePacUsersForReferensi() {
  return getActivePacUsers();
}

function listPath(
  path: string,
  query?: string,
  page = 1,
  limit = 10,
  statusFilter?: string,
  penerimaFilter?: string,
  pacFilter?: string,
  sortKey?: string | null,
  sortDir?: "asc" | "desc",
) {
  return `${path}${queryString({ q: query, page, limit, status: statusFilter, penerima: penerimaFilter, pac: pacFilter, sortKey, sortDir })}`;
}

export async function getPengajuanBerkass(
  query?: string,
  page = 1,
  limit = 10,
  statusFilter?: string,
  penerimaFilter?: string,
  sortKey?: string | null,
  sortDir?: "asc" | "desc",
) {
  return backendRequest<PengajuanPage>(
    listPath(
      "/api/pengajuan",
      query,
      page,
      limit,
      statusFilter,
      penerimaFilter,
      undefined,
      sortKey,
      sortDir,
    ),
  );
}

export async function getVerifikasiPengajuanForCabang(
  query?: string,
  page = 1,
  limit = 10,
  statusFilter?: string,
  penerimaFilter?: string,
  pacFilter?: string,
  sortKey?: string | null,
  sortDir?: "asc" | "desc",
) {
  return backendRequest<PengajuanPage>(
    listPath(
      "/api/pengajuan",
      query,
      page,
      limit,
      statusFilter,
      penerimaFilter,
      pacFilter,
      sortKey,
      sortDir,
    ),
  );
}

export async function getPengajuanForReferensiPac(
  query?: string,
  page = 1,
  limit = 10,
  statusFilter?: string,
  penerimaFilter?: string,
  pacFilter?: string,
  sortKey?: string | null,
  sortDir?: "asc" | "desc",
) {
  return backendRequest<PengajuanPage>(
    listPath(
      "/api/pengajuan/reference",
      query,
      page,
      limit,
      statusFilter,
      penerimaFilter,
      pacFilter,
      sortKey,
      sortDir,
    ),
  );
}

export async function getPengajuanBerkasStatsForReferensi() {
  return backendRequest<PengajuanBerkasStats>("/api/pengajuan/reference/stats");
}

export async function getPengajuanBerkasDetailForReferensi(id: string) {
  return backendDataOrNull<PengajuanBerkas>(
    `/api/pengajuan/reference/${encodeURIComponent(id)}`,
  );
}

async function getDetail(id: string) {
  return backendDataOrNull<PengajuanBerkas>(
    `/api/pengajuan/${encodeURIComponent(id)}`,
  );
}

export async function getPengajuanBerkasById(id: string) {
  return getDetail(id);
}

export async function getPengajuanBerkasDetail(id: string) {
  return getDetail(id);
}

async function save(
  path: string,
  method: "POST" | "PATCH",
  formData: FormData,
) {
  try {
    const result = await backendRequest<MutationResult>(path, {
      method,
      body: formData,
    });
    revalidatePath("/dashboard", "layout");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menyimpan pengajuan.") };
  }
}

export async function createPengajuanBerkas(formData: FormData) {
  return save("/api/pengajuan", "POST", formData);
}

export async function updatePengajuanBerkas(id: string, formData: FormData) {
  return save(`/api/pengajuan/${encodeURIComponent(id)}`, "PATCH", formData);
}

export async function deletePengajuanBerkas(id: string) {
  try {
    const result = await backendRequest<MutationResult>(
      `/api/pengajuan/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    );
    revalidatePath("/dashboard/pengajuan-berkas");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus pengajuan.") };
  }
}

export async function updateStatusPengajuan(
  id: string,
  status: StatusPengajuan,
  alasanPenolakan?: string,
) {
  try {
    const result = await backendRequest<MutationResult>(
      `/api/pengajuan/${encodeURIComponent(id)}/status`,
      {
        method: "PATCH",
        body: { status, alasanPenolakan: alasanPenolakan || "" },
      },
    );
    revalidatePath("/dashboard/pengajuan-berkas");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal mengubah status pengajuan.") };
  }
}

export async function getPengajuanBerkasStats(userId?: string) {
  return backendRequest<PengajuanBerkasStats>(
    `/api/pengajuan/stats${queryString({ userId })}`,
  );
}

export async function getPengajuanDownloadToken(id: string) {
  return (
    await backendRequest<{ token: string }>(
      `/api/pengajuan/${encodeURIComponent(id)}/download-token`,
      { method: "POST" },
    )
  ).token;
}
