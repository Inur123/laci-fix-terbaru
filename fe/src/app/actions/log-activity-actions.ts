"use server";

import { backendDataOrNull, backendRequest, errorMessage, queryString } from "@/lib/backend";

export type LogActivityFilters = {
  search?: string;
  action?: string;
  module?: string;
  startDate?: string;
  endDate?: string;
  periodeId?: string;
  userId?: string;
  sortKey?: string | null;
  sortDir?: "asc" | "desc";
};

export type LogModule = "ARSIP_SURAT" | "ANGGOTA" | "BERKAS_PIMPINAN" | "BERKAS_SP" | "AGENDA_KEGIATAN" | "PENGAJUAN_BERKAS" | "PERIODE" | "USER" | "AUTH" | "PRESENSI";

function logsPath(filters: LogActivityFilters, page: number, limit: number, scope: "personal" | "global" = "personal") {
  return `/api/logs${queryString({
    q: filters.search,
    action: filters.action,
    module: filters.module,
    startDate: filters.startDate,
    endDate: filters.endDate,
    userId: filters.userId,
    sortKey: filters.sortKey,
    sortDir: filters.sortDir,
    page,
    limit,
	scope,
  })}`;
}

export async function getPersonalLogs(filters: Omit<LogActivityFilters, "periodeId"> = {}, page = 1, pageSize = 20) {
  return backendRequest<any>(logsPath(filters, page, pageSize));
}

export async function getGlobalLogs(filters: Omit<LogActivityFilters, "periodeId"> = {}, page = 1, pageSize = 20) {
  return backendRequest<any>(logsPath(filters, page, pageSize, "global"));
}

export async function getLogStats() {
  return backendRequest<Record<string, number>>("/api/logs/stats");
}

export async function getGlobalLogStats(userId?: string) {
  return backendRequest<Record<string, number>>(`/api/logs/stats${queryString({ userId, scope: "global" })}`);
}

export async function getLogMonitoringData(userId?: string) {
  return backendRequest<any>(`/api/logs/monitoring${queryString({ userId })}`);
}

export async function getLogActivityById(id: string) {
  return backendDataOrNull(`/api/logs/${encodeURIComponent(id)}`);
}

async function writeClientLog(module: LogModule, description: string) {
  try {
    return await backendRequest<any>("/api/logs", { method: "POST", body: { action: "EXPORT", module, description } });
  } catch (error) {
    return { error: errorMessage(error, "Gagal mencatat aktivitas.") };
  }
}

export async function logExport(module: LogModule, fileName: string) {
  return writeClientLog(module, `Mengekspor data ${module.replaceAll("_", " ")} ke Excel: ${fileName}`);
}
