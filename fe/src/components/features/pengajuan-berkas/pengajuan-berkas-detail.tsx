"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { updateStatusPengajuan } from "@/app/actions/pengajuan-berkas-actions";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  CheckCircle,
  XCircle,
  Clock,
  FileText,
  User,
  Calendar,
  Building,
  Download,
  AlertCircle,
  ExternalLink,
  Eye,
} from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ConfirmModal } from "@/components/shared/confirm-modal";
import { Spinner } from "@/components/ui/spinner";
import { isPdf, isImage } from "@/lib/file-utils";
import { getPengajuanDownloadToken } from "@/app/actions/pengajuan-berkas-actions";
import { useIsMobile } from "@/hooks/use-mobile";
import { Smartphone } from "lucide-react";
import { useEffect } from "react";
import { capitalizeName } from "@/lib/utils";
import type { PengajuanBerkas } from "@/types/domain";

const MAX_REJECT_REASON_LENGTH = 2000;

const statusConfig = {
  PENDING: {
    label: "Pending",
    icon: Clock,
    className:
      "bg-amber-100/80 text-amber-700 border-amber-200 hover:bg-amber-200/80",
  },
  DITERIMA: {
    label: "Diterima",
    icon: CheckCircle,
    className:
      "bg-green-100/80 text-green-700 border-green-200 hover:bg-green-200/80",
  },
  DITOLAK: {
    label: "Ditolak",
    icon: XCircle,
    className: "bg-red-100/80 text-red-700 border-red-200 hover:bg-red-200/80",
  },
};

const penerimaConfig: Record<string, { label: string; className: string }> = {
  IPNU: {
    label: "IPNU",
    className:
      "bg-emerald-100/80 text-emerald-700 border-emerald-200 hover:bg-emerald-200/80",
  },
  IPPNU: {
    label: "IPPNU",
    className:
      "bg-rose-100/80 text-rose-700 border-rose-200 hover:bg-rose-200/80",
  },
  BERSAMA: {
    label: "BERSAMA",
    className:
      "bg-indigo-100/80 text-indigo-700 border-indigo-200 hover:bg-indigo-200/80",
  },
  CBP_KPP: {
    label: "CBP KPP",
    className:
      "bg-amber-100/80 text-amber-700 border-amber-200 hover:bg-amber-200/80",
  },
};

export function PengajuanBerkasDetail({
  pengajuan,
  isCabang,
  showSubmitterInfo = isCabang,
  showActions = isCabang,
  showFileAccess = true,
}: {
  pengajuan: PengajuanBerkas;
  isCabang: boolean;
  showSubmitterInfo?: boolean;
  showActions?: boolean;
  showFileAccess?: boolean;
}) {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showRejectForm, setShowRejectForm] = useState(false);
  const [alasanPenolakan, setAlasanPenolakan] = useState("");

  const [isPdfLoading, setIsPdfLoading] = useState(true);
  const [isImageLoading, setIsImageLoading] = useState(true);
  const [downloadToken, setDownloadToken] = useState<string | null>(null);
  const isMobile = useIsMobile();

  const showPdfPreview = isPdf(pengajuan.file);
  const showImagePreview = isImage(pengajuan.file);

  const previewUrl = downloadToken
    ? `/api/pengajuan-berkas/download/${pengajuan.id}?preview=true&token=${downloadToken}`
    : undefined;
  const downloadUrl = downloadToken
    ? `/api/pengajuan-berkas/download/${pengajuan.id}?token=${downloadToken}`
    : undefined;

  useEffect(() => {
    if (showFileAccess && pengajuan.id && pengajuan.file) {
      getPengajuanDownloadToken(pengajuan.id)
        .then(setDownloadToken)
        .catch(console.error);
    }
  }, [pengajuan.id, pengajuan.file, showFileAccess]);

  const [confirmApproveOpen, setConfirmApproveOpen] = useState(false);
  const [confirmRejectOpen, setConfirmRejectOpen] = useState(false);

  const StatusIcon =
    statusConfig[pengajuan.status as keyof typeof statusConfig]?.icon || Clock;

  const shortPurpose =
    pengajuan.keperluan.length > 60
      ? `${pengajuan.keperluan.slice(0, 57)}...`
      : pengajuan.keperluan;

  const handleApprove = async () => {
    setIsSubmitting(true);
    const result = await updateStatusPengajuan(pengajuan.id, "DITERIMA");
    setIsSubmitting(false);
    setConfirmApproveOpen(false);

    if (result.error) {
      toast.error(result.error);
    } else {
      toast.success(result.success);
      router.push("/dashboard/pengajuan-berkas");
      router.refresh();
    }
  };

  const handleReject = async () => {
    if (!alasanPenolakan.trim()) {
      toast.error("Alasan penolakan harus diisi");
      return;
    }

    setIsSubmitting(true);
    const result = await updateStatusPengajuan(
      pengajuan.id,
      "DITOLAK",
      alasanPenolakan,
    );
    setIsSubmitting(false);
    setConfirmRejectOpen(false);

    if (result.error) {
      toast.error(result.error);
    } else {
      toast.success(result.success);
      router.push("/dashboard/pengajuan-berkas");
      router.refresh();
    }
  };

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="h-full">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-xl">Informasi Pengajuan</CardTitle>
              <Badge
                variant="outline"
                className={
                  statusConfig[pengajuan.status as keyof typeof statusConfig]
                    ?.className
                }
              >
                <StatusIcon className="w-4 h-4 mr-1" />
                {
                  statusConfig[pengajuan.status as keyof typeof statusConfig]
                    ?.label
                }
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label className="text-muted-foreground flex items-center gap-2">
                  <FileText className="w-4 h-4" />
                  Nomor Surat
                </Label>
                <p className="break-words font-medium [overflow-wrap:anywhere]">
                  {pengajuan.noSurat}
                </p>
              </div>

              <div className="space-y-2">
                <Label className="text-muted-foreground flex items-center gap-2">
                  <Building className="w-4 h-4" />
                  Penerima
                </Label>
                <Badge
                  variant="outline"
                  className={
                    penerimaConfig[pengajuan.penerima]?.className ||
                    "bg-slate-50 text-slate-700 border-slate-200 shadow-none"
                  }
                >
                  {penerimaConfig[pengajuan.penerima]?.label ||
                    pengajuan.penerima}
                </Badge>
              </div>

              <div className="space-y-2">
                <Label className="text-muted-foreground flex items-center gap-2">
                  <Calendar className="w-4 h-4" />
                  Tanggal Surat
                </Label>
                <p className="font-medium">
                  {new Date(pengajuan.tanggal).toLocaleDateString("id-ID", {
                    day: "numeric",
                    month: "long",
                    year: "numeric",
                  })}
                </p>
              </div>

              <div className="space-y-2">
                <Label className="text-muted-foreground flex items-center gap-2">
                  <Calendar className="w-4 h-4" />
                  Tanggal Pengajuan
                </Label>
                <p className="font-medium">
                  {new Date(pengajuan.createdAt).toLocaleDateString("id-ID", {
                    day: "numeric",
                    month: "long",
                    year: "numeric",
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </p>
              </div>
            </div>

            <div className="space-y-2 pt-4 border-t">
              <Label className="text-muted-foreground">Keperluan</Label>
              <p className="break-words font-medium [overflow-wrap:anywhere]">
                {pengajuan.keperluan}
              </p>
            </div>

            {pengajuan.deskripsi && (
              <div className="space-y-2">
                <Label className="text-muted-foreground">Deskripsi</Label>
                <p className="whitespace-pre-wrap break-words text-sm [overflow-wrap:anywhere]">
                  {pengajuan.deskripsi}
                </p>
              </div>
            )}

            {pengajuan.status === "DITOLAK" && pengajuan.alasanPenolakan && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <strong>Alasan Penolakan:</strong>
                  <p className="mt-1 whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
                    {pengajuan.alasanPenolakan}
                  </p>
                </AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>

        <div className="flex flex-col gap-6 h-full">
          {showSubmitterInfo ? (
            <Card className="flex-1">
              <CardHeader>
                <CardTitle className="text-lg">Informasi Pengaju</CardTitle>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex items-center gap-3">
                  <div className="h-10 w-10 rounded-full bg-slate-100 flex items-center justify-center text-slate-500">
                    <User className="w-5 h-5" />
                  </div>
                  <div className="min-w-0">
                    <p className="break-words font-medium [overflow-wrap:anywhere]">
                      {capitalizeName(pengajuan.user.name)}
                    </p>
                    <p className="break-all text-xs text-muted-foreground">
                      {pengajuan.user.email}
                    </p>
                  </div>
                </div>

                <div className="pt-4 border-t grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div>
                    <Label className="text-xs text-muted-foreground">
                      Periode PAC
                    </Label>
                    <p className="break-words text-sm font-medium [overflow-wrap:anywhere]">
                      {pengajuan.periodePac.nama}
                    </p>
                  </div>
                  <div>
                    <Label className="text-xs text-muted-foreground">
                      Periode Cabang
                    </Label>
                    <p className="break-words text-sm font-medium [overflow-wrap:anywhere]">
                      {pengajuan.periodeCabang.nama}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ) : (
            <Card className="flex-1">
              <CardHeader>
                <CardTitle className="text-lg">Status Pengajuan</CardTitle>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex flex-col items-center justify-center p-6 bg-slate-50 rounded-xl border border-dashed gap-4">
                  <div
                    className={`p-4 rounded-full ${
                      pengajuan.status === "PENDING"
                        ? "bg-amber-100/50 text-amber-600"
                        : pengajuan.status === "DITERIMA"
                          ? "bg-green-100/50 text-green-600"
                          : "bg-red-100/50 text-red-600"
                    }`}
                  >
                    <StatusIcon className="w-10 h-10" />
                  </div>
                  <div className="text-center">
                    <p className="font-bold text-lg">
                      {
                        statusConfig[
                          pengajuan.status as keyof typeof statusConfig
                        ]?.label
                      }
                    </p>
                    <p className="text-sm text-muted-foreground mt-1 max-w-[200px]">
                      {pengajuan.status === "PENDING"
                        ? "Menunggu verifikasi dan persetujuan dari Sekretaris Cabang."
                        : pengajuan.status === "DITERIMA"
                          ? "Pengajuan Anda telah disetujui. Silakan cek arsip berkas."
                          : "Pengajuan Anda tidak disetujui. Silakan periksa alasan penolakan."}
                    </p>
                  </div>
                </div>

                <div className="pt-4 border-t space-y-4">
                  <div>
                    <Label className="text-xs text-muted-foreground">
                      Informasi Periode
                    </Label>
                    <div className="grid grid-cols-2 gap-6 mt-2">
                      <div className="p-3 bg-slate-50 rounded-lg border border-slate-100">
                        <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                          PAC
                        </p>
                        <p className="break-words text-xs font-bold [overflow-wrap:anywhere]">
                          {pengajuan.periodePac.nama}
                        </p>
                      </div>
                      <div className="p-3 bg-slate-50 rounded-lg border border-slate-100">
                        <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">
                          Cabang
                        </p>
                        <p className="break-words text-xs font-bold [overflow-wrap:anywhere]">
                          {pengajuan.periodeCabang.nama}
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {showActions && pengajuan.status === "PENDING" && (
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Tindakan</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {!showRejectForm ? (
                  <div className="space-y-3">
                    <Button
                      onClick={() => setConfirmApproveOpen(true)}
                      disabled={isSubmitting}
                      className="min-h-11 w-full bg-green-600 hover:bg-green-700 text-white shadow-md hover:shadow-xl transition-all duration-200"
                    >
                      <CheckCircle className="w-4 h-4 mr-2" />
                      Terima Pengajuan
                    </Button>
                    <Button
                      onClick={() => setShowRejectForm(true)}
                      disabled={isSubmitting}
                      className="min-h-11 w-full bg-red-600 hover:bg-red-700 text-white shadow-md hover:shadow-xl transition-all duration-200"
                    >
                      <XCircle className="w-4 h-4 mr-2" />
                      Tolak Pengajuan
                    </Button>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="alasan">
                        Alasan Penolakan <span className="text-red-500">*</span>
                      </Label>
                      <Textarea
                        id="alasan"
                        value={alasanPenolakan}
                        onChange={(e) => setAlasanPenolakan(e.target.value)}
                        placeholder="Jelaskan alasan penolakan..."
                        rows={4}
                        maxLength={MAX_REJECT_REASON_LENGTH}
                        required
                      />
                    </div>
                    <div className="flex flex-col gap-2">
                      <Button
                        onClick={() => setConfirmRejectOpen(true)}
                        disabled={isSubmitting || !alasanPenolakan.trim()}
                        className="min-h-11 w-full bg-red-600 hover:bg-red-700 text-white shadow-md hover:shadow-xl transition-all duration-200"
                      >
                        Konfirmasi Penolakan
                      </Button>
                      <Button
                        onClick={() => {
                          setShowRejectForm(false);
                          setAlasanPenolakan("");
                        }}
                        disabled={isSubmitting}
                        variant="outline"
                        className="min-h-11 w-full hover:bg-slate-100 transition-all duration-200"
                      >
                        Batal
                      </Button>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {pengajuan.file ? (
        <Card>
          <CardHeader>
            <CardTitle>File Lampiran</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            {showFileAccess ? (
              <div className="flex flex-col md:flex-row items-center justify-between p-4 border rounded-lg gap-4">
                <div className="flex items-center gap-3">
                  <div className="p-3 bg-slate-100 rounded-full">
                    <FileText className="w-8 h-8 text-slate-500" />
                  </div>
                  <p className="font-medium">File Surat (Terenkripsi)</p>
                </div>
                <div className="flex flex-wrap gap-2 w-full md:w-auto">
                  {previewUrl && downloadUrl ? (
                    <>
                      <Button
                        variant="outline"
                        asChild
                        className="min-h-11 flex-1 md:flex-none"
                      >
                        <a
                          href={previewUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          <Eye className="w-4 h-4 mr-2" />
                          Buka
                        </a>
                      </Button>
                      <Button asChild className="min-h-11 flex-1 md:flex-none">
                        <a href={downloadUrl}>
                          <Download className="w-4 h-4 mr-2" />
                          Download
                        </a>
                      </Button>
                    </>
                  ) : (
                    <Button disabled className="min-h-11 w-full md:w-auto">
                      <Spinner className="mr-2 h-4 w-4" />
                      Memuat akses file...
                    </Button>
                  )}
                </div>
              </div>
            ) : (
              <p className="rounded-lg border bg-slate-50 p-4 text-sm text-muted-foreground">
                Lampiran hanya dapat diakses oleh pengaju dan Sekretaris Cabang.
              </p>
            )}

            {/* Preview Section */}
            {showFileAccess &&
              downloadToken &&
              (showPdfPreview || showImagePreview) && (
                <div className="space-y-4">
                  <div className="p-2 border-b bg-slate-50/50 text-xs font-medium flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <FileText className="w-3 h-3 text-primary" />
                      Pratinjau {showPdfPreview ? "PDF" : "Gambar"}
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="min-h-11 px-3 text-xs text-primary hover:text-primary hover:bg-primary/5"
                      asChild
                    >
                      <a
                        href={previewUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        <ExternalLink className="w-3 h-3 mr-1" />
                        Layar Penuh
                      </a>
                    </Button>
                  </div>

                  <div className="w-full bg-slate-50 border rounded-lg overflow-hidden relative">
                    {showPdfPreview && (
                      <div className="w-full min-h-[200px] md:h-[750px] bg-white relative">
                        {isMobile ? (
                          <div className="flex flex-col items-center justify-center p-12 text-center bg-slate-50/50 h-full">
                            <div className="p-4 bg-white rounded-full shadow-sm mb-4">
                              <Smartphone className="w-8 h-8 text-primary" />
                            </div>
                            <h3 className="text-sm font-semibold text-slate-800 mb-2">
                              Pratinjau PDF di Mobile
                            </h3>
                            <p className="text-xs text-slate-500 mb-6 max-w-[240px] leading-relaxed">
                              Browser mobile tidak dapat menampilkan PDF secara
                              langsung. Klik tombol di bawah untuk membuka file.
                            </p>
                            <Button
                              asChild
                              className="shadow-md hover:shadow-lg transition-all"
                            >
                              <a
                                href={previewUrl}
                                target="_blank"
                                rel="noopener noreferrer"
                              >
                                <ExternalLink className="w-4 h-4 mr-2" />
                                Buka PDF Sekarang
                              </a>
                            </Button>
                          </div>
                        ) : (
                          <>
                            {isPdfLoading && (
                              <div className="absolute inset-0 flex flex-col items-center justify-center bg-slate-50/50 z-10">
                                <Spinner className="h-8 w-8 mb-3 text-primary" />
                                <p className="text-sm text-muted-foreground animate-pulse">
                                  Memuat...
                                </p>
                              </div>
                            )}
                            <object
                              data={
                                previewUrl
                                  ? `${previewUrl}#view=FitH`
                                  : undefined
                              }
                              type="application/pdf"
                              width="100%"
                              height="100%"
                              className="w-full h-full"
                              onLoad={() => setIsPdfLoading(false)}
                            >
                              <iframe
                                src={previewUrl}
                                className="w-full h-full border-none"
                                title="PDF Preview"
                              />
                            </object>
                          </>
                        )}
                      </div>
                    )}

                    {showImagePreview && (
                      <div className="w-full flex justify-center p-4 relative min-h-[300px]">
                        {isImageLoading && (
                          <div className="absolute inset-0 flex flex-col items-center justify-center bg-slate-50/50 z-10">
                            <Spinner className="h-8 w-8 mb-3 text-primary" />
                            <p className="text-sm text-muted-foreground animate-pulse">
                              Memuat...
                            </p>
                          </div>
                        )}
                        <img
                          src={previewUrl}
                          alt="Pratinjau Gambar"
                          className={`max-w-full max-h-[1000px] rounded shadow-sm object-contain transition-opacity duration-300 ${isImageLoading ? "opacity-0" : "opacity-100"}`}
                          onLoad={() => setIsImageLoading(false)}
                        />
                      </div>
                    )}
                  </div>
                </div>
              )}

            {showFileAccess && (
              <p className="text-xs text-muted-foreground mt-2 flex items-center gap-1">
                <AlertCircle className="w-3 h-3" />
                File siap diakses secara otomatis saat dibuka atau diunduh
              </p>
            )}
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>File Lampiran</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col items-center justify-center p-8 border-2 border-dashed rounded-lg text-center bg-slate-50">
              <div className="p-4 bg-slate-100 rounded-full mb-3">
                <FileText className="w-8 h-8 text-muted-foreground/50" />
              </div>
              <h3 className="font-medium text-muted-foreground">
                File tidak ada
              </h3>
              <p className="text-sm text-muted-foreground/70 mt-1">
                Pengajuan ini tidak memiliki lampiran file.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      <ConfirmModal
        isOpen={confirmApproveOpen}
        onClose={() => setConfirmApproveOpen(false)}
        onConfirm={handleApprove}
        title="Terima Pengajuan?"
        description={`Apakah Anda yakin ingin menerima pengajuan "${shortPurpose}" dari ${capitalizeName(pengajuan.user.name)}?`}
        confirmText="Ya, Terima"
        variant="default"
        loading={isSubmitting}
      />

      <ConfirmModal
        isOpen={confirmRejectOpen}
        onClose={() => setConfirmRejectOpen(false)}
        onConfirm={handleReject}
        title="Tolak Pengajuan?"
        description={`Apakah Anda yakin ingin menolak pengajuan "${shortPurpose}" dari ${capitalizeName(pengajuan.user.name)}? Alasan penolakan akan dikirim ke pengaju.`}
        confirmText="Ya, Tolak"
        variant="destructive"
        loading={isSubmitting}
      />
    </div>
  );
}
