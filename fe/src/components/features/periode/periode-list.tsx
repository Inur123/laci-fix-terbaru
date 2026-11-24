"use client";

import type { Periode } from "@/types/domain";
import {
  activatePeriode,
  deletePeriode,
  getPeriodes,
} from "@/app/actions/periode-actions";
import { setViewPeriode } from "@/app/actions/view-periode-actions";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { useCallback, useEffect, useRef, useState } from "react";
import { Trash2, CheckCircle, Edit, Info, Loader2, Eye } from "lucide-react";
import Link from "next/link";
import { ConfirmModal } from "@/components/shared/confirm-modal";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { cn } from "@/lib/utils";

export function PeriodeList({
  periods,
  userRole,
  activeViewId
}: {
  periods: Periode[];
  userRole?: string;
  activeViewId?: string;
}) {
  const [data, setData] = useState<Periode[]>(periods);
  const [loadingId, setLoadingId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [viewLoadingId, setViewLoadingId] = useState<string | null>(null);
  const realtimeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setData(periods);
  }, [periods]);

  const refreshData = useCallback(async () => {
    try {
      setData(await getPeriodes());
    } catch {
      toast.error("Gagal memperbarui daftar periode.");
    }
  }, []);

  async function handleActivate(id: string) {
    setLoadingId(id);
    try {
      const result = await activatePeriode(id);
      if (result.error) toast.error(result.error);
      else {
        toast.success("Periode berhasil diaktifkan!");
        await refreshData();
      }
    } finally {
      setLoadingId(null);
    }
  }

  async function handleView(id: string, name: string) {
    setViewLoadingId(id);
    try {
      const result = await setViewPeriode(id);
      if (result.success) {
        toast.success(`Berhasil beralih ke periode data ${name}!`);
        // Refresh halaman agar state server di-render ulang
        window.location.reload();
      }
    } catch (err) {
      console.error(err);
      toast.error("Gagal beralih periode data");
    } finally {
      setViewLoadingId(null);
    }
  }

  async function handleDelete() {
    if (!confirmDeleteId) return;
    const id = confirmDeleteId;
    setLoadingId(id);
    try {
      const result = await deletePeriode(id);
      if (result.error) {
        toast.error(result.error);
      } else {
        setConfirmDeleteId(null);
        toast.success("Periode berhasil dihapus!");
        await refreshData();
      }
    } finally {
      setLoadingId(null);
    }
  }

  useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent).detail as {
        type?: string;
        model?: string;
      };
      if (!detail || detail.type !== "mutation") return;
      if (detail.model !== "Periode") return;
      if (realtimeTimerRef.current) return;
      realtimeTimerRef.current = setTimeout(() => {
        realtimeTimerRef.current = null;
        refreshData();
      }, 300);
    };
    window.addEventListener("laci-realtime", handler as EventListener);
    return () => {
      window.removeEventListener("laci-realtime", handler as EventListener);
      if (realtimeTimerRef.current) {
        clearTimeout(realtimeTimerRef.current);
        realtimeTimerRef.current = null;
      }
    };
  }, [refreshData]);

  if (data.length === 0) {
    return (
      <Alert>
        <Info className="h-4 w-4" />
        <AlertTitle>Informasi</AlertTitle>
        <AlertDescription>Belum ada periode yang dibuat.</AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="grid gap-4">
      {data.map((periode) => {
        const isViewing = Boolean(
          (activeViewId && activeViewId === periode.id) ||
            (!activeViewId && periode.isActive),
        );
        const formattedDate = new Date(periode.createdAt).toLocaleDateString(
          "id-ID",
          {
            day: "numeric",
            month: "long",
            year: "numeric",
          },
        );

        return (
          <div
            key={periode.id}
            className={cn(
              "flex flex-col gap-3 p-4 sm:p-5 rounded-xl border bg-card shadow-sm transition-all sm:flex-row sm:items-center sm:justify-between sm:gap-4",
              periode.isActive
                ? userRole === "SEKRETARIS_CABANG"
                  ? "border-blue-400 ring-1 ring-blue-400/20"
                  : "border-emerald-400 ring-1 ring-emerald-400/20"
                : "border-slate-200",
            )}
          >
            {/* Header Info */}
            <div className="flex items-start justify-between sm:block gap-2 min-w-0">
              <div className="min-w-0 space-y-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-lg font-bold text-slate-900 tracking-tight break-words">
                    {periode.nama}
                  </span>
                  {periode.isActive && (
                    <Badge
                      variant="outline"
                      className={cn(
                        "shrink-0 font-semibold text-xs shadow-none",
                        userRole === "SEKRETARIS_CABANG"
                          ? "bg-blue-100/80 text-blue-700 border-blue-200"
                          : "bg-emerald-100/80 text-emerald-700 border-emerald-200",
                      )}
                    >
                      Aktif
                    </Badge>
                  )}
                </div>
                <p className="text-xs text-slate-500">Dibuat: {formattedDate}</p>
              </div>

              {/* Quick Actions on Mobile (Top Right) */}
              <div className="flex items-center gap-1 sm:hidden shrink-0">
                <Button
                  size="sm"
                  variant="outline"
                  className="h-8 px-2.5 text-xs bg-white text-slate-700 border-slate-200 shadow-sm"
                  asChild
                >
                  <Link href={`/dashboard/periode/${periode.id}/edit`}>
                    <Edit className="w-3.5 h-3.5 mr-1 text-slate-500" />
                    Edit
                  </Link>
                </Button>
                {!periode.isActive && (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-8 w-8 p-0 text-red-500 hover:text-red-600 hover:bg-red-50"
                    onClick={() => setConfirmDeleteId(periode.id)}
                    disabled={loadingId === periode.id}
                    title="Hapus periode"
                    aria-label={`Hapus periode ${periode.nama}`}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                )}
              </div>
            </div>

            {/* Actions Bar (Bottom on Mobile, Right on Desktop) */}
            <div className="flex items-center gap-2 w-full pt-2.5 sm:pt-0 border-t sm:border-t-0 border-slate-100 sm:w-auto sm:justify-end">
              <Button
                size="sm"
                variant={isViewing ? "default" : "outline"}
                className={cn(
                  "flex-1 sm:flex-initial h-9 text-xs sm:text-sm font-medium whitespace-nowrap",
                  isViewing
                    ? userRole === "SEKRETARIS_CABANG"
                      ? "bg-blue-600 hover:bg-blue-700 text-white shadow-sm"
                      : "bg-green-600 hover:bg-green-700 text-white shadow-sm"
                    : "bg-white",
                )}
                disabled={viewLoadingId === periode.id}
                onClick={() => handleView(periode.id, periode.nama)}
              >
                {viewLoadingId === periode.id ? (
                  <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
                ) : (
                  <Eye className="w-3.5 h-3.5 mr-1.5" />
                )}
                {isViewing ? "Sedang Ditampilkan" : "Tampilkan"}
              </Button>

              {!periode.isActive && (
                <Button
                  size="sm"
                  variant="outline"
                  className="flex-1 sm:flex-initial h-9 text-xs sm:text-sm font-medium bg-white whitespace-nowrap"
                  onClick={() => handleActivate(periode.id)}
                  disabled={loadingId === periode.id}
                >
                  {loadingId === periode.id ? (
                    <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
                  ) : (
                    <CheckCircle
                      className={cn(
                        "w-3.5 h-3.5 mr-1.5",
                        userRole === "SEKRETARIS_CABANG"
                          ? "text-blue-600"
                          : "text-emerald-600",
                      )}
                    />
                  )}
                  {loadingId === periode.id ? "Memproses..." : "Aktifkan"}
                </Button>
              )}

              {/* Desktop Actions (Edit & Delete) */}
              <div className="hidden sm:flex sm:items-center sm:gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  className="h-9 px-3 text-sm bg-white"
                  asChild
                >
                  <Link href={`/dashboard/periode/${periode.id}/edit`}>
                    <Edit className="w-3.5 h-3.5 mr-1.5 text-slate-500" />
                    Edit
                  </Link>
                </Button>
                {!periode.isActive && (
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-9 w-9 p-0 text-red-500 hover:text-red-600 hover:bg-red-50"
                    onClick={() => setConfirmDeleteId(periode.id)}
                    disabled={loadingId === periode.id}
                    title="Hapus periode"
                    aria-label={`Hapus periode ${periode.nama}`}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                )}
              </div>
            </div>
          </div>
        );
      })}

      <ConfirmModal
        isOpen={!!confirmDeleteId}
        onClose={() => {
          if (!loadingId) setConfirmDeleteId(null);
        }}
        onConfirm={handleDelete}
        title="Hapus Periode?"
        description="Apakah Anda yakin ingin menghapus periode ini? Tindakan ini tidak dapat dibatalkan."
        variant="destructive"
        loading={!!loadingId}
      />
    </div>
  );
}
