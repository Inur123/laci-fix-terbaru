"use client";

import { useState } from "react";
import {
  ArrowLeft,
  Calendar,
  Clock,
  MapPin,
  Pencil,
  Trash2,
  CheckCircle,
  PlayCircle,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { deleteKegiatan, type KegiatanItem } from "@/app/actions/agenda-kegiatan-actions";
import { ConfirmModal } from "@/components/shared/confirm-modal";
import { toast } from "sonner";
import { capitalizeName, cn } from "@/lib/utils";

const statusConfig: Record<
  string,
  { label: string; icon: LucideIcon; className: string }
> = {
  MENDATANG: {
    label: "Mendatang",
    icon: Clock,
    className:
      "bg-blue-100/80 text-blue-700 border-blue-200 hover:bg-blue-200/80",
  },
  BERLANGSUNG: {
    label: "Berlangsung",
    icon: PlayCircle,
    className:
      "bg-green-100/80 text-green-700 border-green-200 hover:bg-green-200/80",
  },
  SELESAI: {
    label: "Selesai",
    icon: CheckCircle,
    className:
      "bg-slate-100/80 text-slate-700 border-slate-200 hover:bg-slate-200/80",
  },
};

function computeStatus(tanggalMulai: Date | string, tanggalSelesai?: Date | string | null): string {
  const now = new Date();
  const start = new Date(tanggalMulai);
  const end = tanggalSelesai ? new Date(tanggalSelesai) : start;

  if (now < start) {
    return "MENDATANG";
  } else if (now > end) {
    return "SELESAI";
  } else {
    return "BERLANGSUNG";
  }
}

export function KegiatanDetail({
  kegiatan,
  userRole,
}: {
  kegiatan: KegiatanItem;
  userRole?: string;
}) {
  const router = useRouter();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const isCabang = userRole === "SEKRETARIS_CABANG";
  const status = computeStatus(kegiatan.tanggalMulai, kegiatan.tanggalSelesai);
  const StatusIcon = statusConfig[status]?.icon || Clock;

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      const result = await deleteKegiatan(kegiatan.id);
      if (result.error) {
        toast.error(result.error);
        setIsDeleting(false);
      } else {
        toast.success("Kegiatan berhasil dihapus!");
        router.push("/dashboard/agenda-kegiatan");
        router.refresh();
      }
    } catch {
      toast.error("Terjadi kesalahan saat menghapus kegiatan");
      setIsDeleting(false);
    }
  };

  return (
    <div className="w-full flex flex-col gap-4 sm:gap-6">
      {/* Header Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <Button
            variant="outline"
            size="icon"
            asChild
            className="h-11 w-11 sm:h-9 sm:w-9"
          >
            <Link
              href="/dashboard/agenda-kegiatan"
              aria-label="Kembali ke Agenda Kegiatan"
            >
              <ArrowLeft className="w-4 h-4" />
            </Link>
          </Button>
          <div>
            <h2 className="text-xl font-semibold tracking-tight">Detail Kegiatan</h2>
            <p className="text-sm text-muted-foreground">
              Informasi lengkap jadwal dan agenda kegiatan
            </p>
          </div>
        </div>

        {isCabang && (
          <div className="flex items-center gap-2 self-end sm:self-auto">
            <Button
              variant="outline"
              size="sm"
              className="h-11 px-4 sm:h-9 text-xs sm:text-sm font-medium"
              asChild
            >
              <Link href={`/dashboard/agenda-kegiatan/${kegiatan.id}/edit`}>
                <Pencil className="w-3.5 h-3.5 mr-1.5" />
                Edit
              </Link>
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-11 px-4 sm:h-9 text-xs sm:text-sm font-medium text-red-500 hover:text-red-600 hover:bg-red-50"
              onClick={() => setConfirmDelete(true)}
            >
              <Trash2 className="w-3.5 h-3.5 mr-1.5" />
              Hapus
            </Button>
          </div>
        )}
      </div>

      {/* Main Info Card */}
      <Card className="shadow-sm border-slate-200">
        <CardHeader className="p-5 sm:p-6 border-b border-slate-100">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <div
                className="w-4 h-4 rounded-full shrink-0 shadow-sm"
                style={{ backgroundColor: kegiatan.warna || "#3b82f6" }}
                title={`Warna label: ${kegiatan.warna}`}
              />
              <CardTitle className="text-lg sm:text-xl font-bold text-slate-900 break-words">
                {capitalizeName(kegiatan.judul)}
              </CardTitle>
            </div>
            <Badge
              variant="outline"
              className={cn(
                "transition-colors flex w-fit items-center gap-1.5 font-medium px-2.5 py-1",
                statusConfig[status]?.className,
              )}
            >
              <StatusIcon size={14} />
              {statusConfig[status]?.label || status}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="p-5 sm:p-6 space-y-6">
          {/* Key Details Grid */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6 bg-slate-50/60 p-4 sm:p-5 rounded-xl border border-slate-100">
            {/* Waktu Mulai */}
            <div className="space-y-1">
              <span className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Calendar className="w-3.5 h-3.5 text-slate-400" />
                Waktu Mulai
              </span>
              <p className="text-sm font-semibold text-slate-900">
                {new Date(kegiatan.tanggalMulai).toLocaleDateString("id-ID", {
                  weekday: "long",
                  day: "numeric",
                  month: "long",
                  year: "numeric",
                })}
              </p>
              <p className="text-xs text-slate-500">
                Pukul{" "}
                {new Date(kegiatan.tanggalMulai).toLocaleTimeString("id-ID", {
                  hour: "2-digit",
                  minute: "2-digit",
                })}{" "}
                WIB
              </p>
            </div>

            {/* Waktu Selesai */}
            <div className="space-y-1">
              <span className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <Clock className="w-3.5 h-3.5 text-slate-400" />
                Waktu Selesai
              </span>
              {kegiatan.tanggalSelesai ? (
                <>
                  <p className="text-sm font-semibold text-slate-900">
                    {new Date(kegiatan.tanggalSelesai).toLocaleDateString("id-ID", {
                      weekday: "long",
                      day: "numeric",
                      month: "long",
                      year: "numeric",
                    })}
                  </p>
                  <p className="text-xs text-slate-500">
                    Pukul{" "}
                    {new Date(kegiatan.tanggalSelesai).toLocaleTimeString("id-ID", {
                      hour: "2-digit",
                      minute: "2-digit",
                    })}{" "}
                    WIB
                  </p>
                </>
              ) : (
                <p className="text-sm text-slate-500 italic">Tidak ditentukan</p>
              )}
            </div>

            {/* Lokasi */}
            <div className="space-y-1">
              <span className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
                <MapPin className="w-3.5 h-3.5 text-slate-400" />
                Lokasi Kegiatan
              </span>
              <p className="text-sm font-semibold text-slate-900">
                {kegiatan.lokasi ? capitalizeName(kegiatan.lokasi) : "Tidak ada keterangan lokasi"}
              </p>
            </div>
          </div>

          {/* Deskripsi Section */}
          <div className="space-y-2">
            <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Deskripsi Kegiatan
            </h4>
            <div className="p-4 rounded-xl bg-white border border-slate-200">
              {kegiatan.deskripsi ? (
                <p className="text-sm text-slate-700 whitespace-pre-line leading-relaxed">
                  {kegiatan.deskripsi}
                </p>
              ) : (
                <p className="text-sm text-muted-foreground italic">
                  Tidak ada deskripsi tambahan untuk kegiatan ini.
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={handleDelete}
        title="Hapus Kegiatan?"
        description={`Apakah Anda yakin ingin menghapus kegiatan "${capitalizeName(kegiatan.judul)}"? Tindakan ini tidak dapat dibatalkan.`}
        variant="destructive"
        loading={isDeleting}
      />
    </div>
  );
}
