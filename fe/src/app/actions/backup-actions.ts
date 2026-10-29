"use server";

import { revalidatePath } from "next/cache";
import { backendRequest, errorMessage, queryString } from "@/lib/backend";

export type BackupItem = { key: string; filename: string; size: number; lastModified: Date };

export async function getBackupListInternal(): Promise<BackupItem[]> {
  const result = await backendRequest<any>("/api/backups");
  return (result.data ?? []).map((item: any) => ({ ...item, lastModified: new Date(item.lastModified) }));
}

export async function getBackupList() {
  return getBackupListInternal();
}

export async function executeBackupLogic(_isCron = false) {
  try {
    return await backendRequest<any>("/api/backups", { method: "POST" });
  } catch (error) {
    return { error: errorMessage(error, "Gagal membuat backup database.") };
  }
}

export async function createDatabaseBackup() {
  return executeBackupLogic(false);
}

export async function deleteDatabaseBackup(key: string) {
  try {
    const result = await backendRequest<any>(`/api/backups${queryString({ key })}`, { method: "DELETE" });
    revalidatePath("/dashboard/backup");
    return result;
  } catch (error) {
    return { error: errorMessage(error, "Gagal menghapus backup database.") };
  }
}

export async function getBackupDownloadUrl(key: string) {
  try {
    return await backendRequest<any>(`/api/backups/download${queryString({ key })}`);
  } catch (error) {
    return { error: errorMessage(error, "Gagal mendapatkan URL download.") };
  }
}
