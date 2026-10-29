"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Progress } from "@/components/ui/progress";
import { Database, Download, Trash2, AlertCircle, Loader2 } from "lucide-react";
import {
  createDatabaseBackup,
  deleteDatabaseBackup,
  getBackupDownloadUrl,
  BackupItem,
} from "@/app/actions/backup-actions";
import { toast } from "sonner";
import { ConfirmModal } from "@/components/shared/confirm-modal";
import { cn } from "@/lib/utils";

export function BackupClient({
  initialBackups,
}: {
  initialBackups: BackupItem[];
}) {
  const [backups, setBackups] = useState<BackupItem[]>(initialBackups);
  const [isBackingUp, setIsBackingUp] = useState(false);
  const [deletingKey, setDeletingKey] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const totalBackups = backups.length;
  const backupLimit = 10;
  const percentUsed = (totalBackups / backupLimit) * 100;

  // Format file size
  const formatSize = (bytes: number) => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  // Format Date
  const formatDate = (date: Date) => {
    const d = new Date(date);
    const datePart = d.toLocaleDateString("id-ID", {
      day: "numeric",
      month: "long",
      year: "numeric",
    });
    const timePart = d.toLocaleTimeString("id-ID", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
    return `${datePart} | ${timePart.replace(/\./g, ":")}`;
  };

  const handleBackup = async () => {
    setIsBackingUp(true);
    try {
      const res = await createDatabaseBackup();
      if (res.error) {
        toast.error(res.error);
      } else {
        toast.success(`Backup berhasil: ${res.filename}`);
        await fetchUpdatedList();
      }
    } catch {
      toast.error("Gagal melakukan backup database");
    } finally {
      setIsBackingUp(false);
    }
  };

  const fetchUpdatedList = async () => {
    try {
      const { getBackupList } = await import("@/app/actions/backup-actions");
      const list = await getBackupList();
      setBackups(list);
    } catch {
      toast.error("Gagal memperbarui daftar backup. Coba lagi beberapa saat.");
    }
  };

  const handleDownload = async (key: string) => {
    try {
      const res = await getBackupDownloadUrl(key);
      if (res.error) {
        toast.error(res.error);
        return;
      }
      if (res.url) {
        window.location.assign(res.url);
      }
    } catch {
      toast.error("Gagal mendownload backup");
    }
  };

  const handleDelete = async () => {
    if (!deletingKey) return;
    setIsDeleting(true);
    const key = deletingKey;
    setDeletingKey(null);

    toast.promise(deleteDatabaseBackup(key), {
      loading: "Menghapus file backup dari Cloudflare R2...",
      success: (res) => {
        if (res.error) throw new Error(res.error);
        fetchUpdatedList();
        return "Backup database berhasil dihapus!";
      },
      error: (err) => err.message || "Gagal menghapus backup",
      finally: () => setIsDeleting(false),
    });
  };

  return (
    <div className="space-y-6">
      {/* Header Section */}
      <div className="flex items-center gap-3">
        <div className="p-2 bg-slate-100 rounded-lg text-slate-500">
          <Database size={24} />
        </div>
        <div>
          <h2 className="text-xl font-semibold">Backup Database</h2>
          <p className="text-sm text-muted-foreground">
            Cadangkan data aplikasi secara manual dan simpan di Cloudflare R2.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Backup Status and Actions */}
        <Card className="md:col-span-1 h-fit">
          <CardHeader>
            <CardTitle className="text-base font-bold flex items-center gap-2">
              Status Penyimpanan R2
            </CardTitle>
            <CardDescription>
              Batas maksimal penyimpanan adalah 10 file backup.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-2">
              <div className="flex justify-between text-xs font-semibold">
                <span>Kapasitas Digunakan</span>
                <span className="text-slate-600 dark:text-slate-400">
                  {totalBackups} dari {backupLimit} Backup
                </span>
              </div>
              <Progress value={percentUsed} className="h-2" />
              {totalBackups >= backupLimit && (
                <div className="flex items-start gap-1.5 p-2.5 bg-amber-50 border border-amber-200 rounded-md text-[11px] text-amber-800">
                  <AlertCircle size={14} className="shrink-0 mt-0.5" />
                  <span>
                    Kapasitas penuh. Backup baru berikutnya akan otomatis
                    menghapus file backup terlama.
                  </span>
                </div>
              )}
            </div>

            <Button
              className="w-full text-white bg-blue-600 hover:bg-blue-700 font-semibold shadow-sm hover:shadow-md transition-all duration-200"
              onClick={handleBackup}
              disabled={isBackingUp || isDeleting}
            >
              {isBackingUp ? (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <Database className="w-4 h-4 mr-2" />
              )}
              {isBackingUp ? "Sedang Membackup..." : "Mulai Backup Database"}
            </Button>
          </CardContent>
        </Card>

        {/* Backup List Table */}
        <Card className="md:col-span-2 ">
          <CardHeader>
            <CardTitle className="text-base font-bold">
              Daftar Backup Aktif
            </CardTitle>
            <CardDescription>
              Backup baru disimpan terenkripsi (.dump.enc). Saat diunduh, file
              menjadi .dump untuk pemulihan PostgreSQL.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0 sm:p-6">
            <div className="rounded-md border overflow-hidden">
              <Table>
                <TableHeader className="bg-slate-50/40 text-slate-500 font-semibold border-b">
                  <TableRow>
                    <TableHead className="w-[50px] text-center bg-slate-50/40 text-slate-500 font-semibold h-11">
                      No
                    </TableHead>
                    <TableHead className="bg-slate-50/40 text-slate-500 font-semibold h-11">
                      Nama File
                    </TableHead>
                    <TableHead className="hidden sm:table-cell bg-slate-50/40 text-slate-500 font-semibold h-11">
                      Waktu Backup
                    </TableHead>
                    <TableHead className="w-[100px] text-right bg-slate-50/40 text-slate-500 font-semibold h-11">
                      Ukuran
                    </TableHead>
                    <TableHead className="w-[100px] text-right bg-slate-50/40 text-slate-500 font-semibold h-11">
                      Aksi
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {backups.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={5}
                        className="h-32 text-center text-muted-foreground"
                      >
                        Belum ada backup database yang dibuat.
                      </TableCell>
                    </TableRow>
                  ) : (
                    backups.map((item, index) => (
                      <TableRow key={item.key} className="hover:bg-slate-50/50">
                        <TableCell className="text-center font-medium text-slate-500">
                          {index + 1}
                        </TableCell>
                        <TableCell className="min-w-0 font-medium">
                          <div className="flex min-w-0 flex-col">
                            <span
                              className="block max-w-[120px] truncate text-sm font-semibold text-slate-800 sm:max-w-[220px]"
                              title={item.filename}
                            >
                              {item.filename}
                            </span>
                            <span className="mt-0.5 max-w-[120px] truncate text-[10px] text-slate-400 sm:hidden">
                              {formatDate(item.lastModified)}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="hidden sm:table-cell text-xs text-slate-600 font-medium whitespace-nowrap">
                          {formatDate(item.lastModified)}
                        </TableCell>
                        <TableCell className="text-right text-xs font-semibold text-slate-700">
                          {formatSize(item.size)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            {item.filename.endsWith(".dump.enc") ? (
                              <Button
                                size="sm"
                                variant="ghost"
                                className="h-11 w-11 p-0 text-slate-500 transition-colors hover:text-slate-900 sm:h-8 sm:w-8"
                                asChild
                              >
                                <a
                                  href={`/api/backups/file?key=${encodeURIComponent(item.key)}`}
                                  className={cn(
                                    (isBackingUp || isDeleting) &&
                                      "pointer-events-none opacity-50",
                                  )}
                                  aria-disabled={isBackingUp || isDeleting}
                                  aria-label={`Download ${item.filename}`}
                                  title={`Download ${item.filename}`}
                                >
                                  <Download className="w-4 h-4" />
                                </a>
                              </Button>
                            ) : (
                              <Button
                                size="sm"
                                variant="ghost"
                                className="h-11 w-11 p-0 text-slate-500 transition-colors hover:text-slate-900 sm:h-8 sm:w-8"
                                onClick={() => handleDownload(item.key)}
                                aria-label={`Download ${item.filename}`}
                                title={`Download ${item.filename}`}
                                disabled={isBackingUp || isDeleting}
                              >
                                <Download className="w-4 h-4" />
                              </Button>
                            )}
                            <Button
                              size="sm"
                              variant="ghost"
                              className="h-11 w-11 p-0 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/20 sm:h-8 sm:w-8"
                              onClick={() => setDeletingKey(item.key)}
                              aria-label={`Hapus ${item.filename}`}
                              title={`Hapus ${item.filename}`}
                              disabled={isBackingUp || isDeleting}
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      </div>

      <ConfirmModal
        isOpen={deletingKey !== null}
        onClose={() => setDeletingKey(null)}
        onConfirm={handleDelete}
        title="Hapus Backup Database?"
        description="Tindakan ini akan menghapus file backup secara permanen dari Cloudflare R2 dan tidak dapat dipulihkan kembali."
        confirmText="Hapus Permanen"
        cancelText="Batal"
      />
    </div>
  );
}
