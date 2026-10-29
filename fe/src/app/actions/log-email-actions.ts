"use server";

import { backendRequest, errorMessage, queryString } from "@/lib/backend";

export interface EmailLogFilters {
  type?: string;
  status?: string;
  search?: string;
  dateFrom?: string;
  dateTo?: string;
  sortKey?: string | null;
  sortDir?: "asc" | "desc";
}

export async function getEmailStats() {
  const result = await backendRequest<any>("/api/email-logs/stats");
  return {
    totalAll: result.totalAll ?? 0,
    totalToday: result.totalToday ?? 0,
    totalSent: result.totalSent ?? 0,
    totalFailed: result.totalFailed ?? 0,
    byType: {
      VERIFICATION: result.verification ?? 0,
      VERIFIED_SUCCESS: result.verifiedSuccess ?? 0,
      PENGAJUAN_USER: result.pengajuanUser ?? 0,
      PENGAJUAN_ADMIN: result.pengajuanAdmin ?? 0,
      PENGAJUAN_STATUS: result.pengajuanStatus ?? 0,
    },
  };
}

export async function getEmailLogs(filters: EmailLogFilters = {}, page = 1, perPage = 20) {
  return backendRequest<any>(`/api/email-logs${queryString({
    type: filters.type,
    status: filters.status,
    q: filters.search,
    dateFrom: filters.dateFrom,
    dateTo: filters.dateTo,
    sortKey: filters.sortKey,
    sortDir: filters.sortDir,
    page,
    limit: perPage,
  })}`);
}

export async function retryEmail(logId: string) {
  try {
    return await backendRequest<any>(`/api/email-logs/${encodeURIComponent(logId)}/retry`, { method: "POST" });
  } catch (error) {
    return { success: false, error: errorMessage(error, "Gagal mengirim ulang email.") };
  }
}

export async function resendVerificationOTP(email: string) {
  try {
    await backendRequest("/api/auth/otp/send", { method: "POST", body: { email } });
    return { success: true };
  } catch (error) {
    return { success: false, error: errorMessage(error, "Gagal mengirim kode OTP.") };
  }
}
