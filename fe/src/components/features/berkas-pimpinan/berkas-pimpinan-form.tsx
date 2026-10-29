"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import {
  createBerkasPimpinan,
  updateBerkasPimpinan,
} from "@/app/actions/berkas-pimpinan-actions";
import {
  Upload,
  X,
  FileText,
  AlertCircle,
  Eye,
  ExternalLink,
} from "lucide-react";
import { Spinner } from "@/components/ui/spinner";
import { DatePicker } from "@/components/ui/date-picker";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { isPdf, isImage } from "@/lib/file-utils";
import { getBerkasPimpinanDownloadToken } from "@/app/actions/berkas-pimpinan-actions";
import { useIsMobile } from "@/hooks/use-mobile";
import { Smartphone } from "lucide-react";

type BerkasPimpinanFormProps = {
  berkas?: {
    id: string;
    nama: string;
    tanggal: Date;
    catatan: string | null;
    file: string | null;
  };
  userRole?: string;
};

const MAX_FILE_SIZE = 5 * 1024 * 1024;
const ALLOWED_FILE_TYPES = [
  "application/pdf",
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.ms-powerpoint",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "image/jpeg",
  "image/png",
  "image/webp",
];

const ALLOWED_EXTENSIONS = [
  ".pdf",
  ".doc",
  ".docx",
  ".ppt",
  ".pptx",
  ".jpg",
  ".jpeg",
  ".png",
  ".webp",
];

export function BerkasPimpinanForm({
  berkas,
  userRole,
}: BerkasPimpinanFormProps) {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [tanggal, setTanggal] = useState<Date | undefined>(
    berkas?.tanggal ? new Date(berkas.tanggal) : undefined,
  );

  const [isPdfLoading, setIsPdfLoading] = useState(true);
  const [isImageLoading, setIsImageLoading] = useState(true);
  const [isDragging, setIsDragging] = useState(false);
  const [downloadToken, setDownloadToken] = useState<string | null>(null);
  const isMobile = useIsMobile();

  useEffect(() => {
    if (berkas?.id && berkas.file) {
      getBerkasPimpinanDownloadToken(berkas.id)
        .then(setDownloadToken)
        .catch(console.error);
    }
  }, [berkas?.id, berkas?.file]);

  const validateFile = (file: File): string | null => {
    if (file.size > MAX_FILE_SIZE) {
      return "Ukuran file maksimal 5MB";
    }

    const fileExtension = "." + file.name.split(".").pop()?.toLowerCase();
    if (!ALLOWED_EXTENSIONS.includes(fileExtension)) {
      return "Format file harus PDF, Word, PowerPoint, atau Gambar (JPG/PNG/WebP)";
    }

    if (!ALLOWED_FILE_TYPES.includes(file.type) && file.type !== "") {
      return "Format file harus PDF, Word, PowerPoint, atau Gambar (JPG/PNG/WebP)";
    }

    return null;
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];

    if (file) {
      const error = validateFile(file);
      if (error) {
        toast.error(error);
        setSelectedFile(null);
        e.target.value = "";
      } else {
        setSelectedFile(file);
      }
    }
  };

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    setIsDragging(false);

    const file = e.dataTransfer.files?.[0];
    if (file) {
      const error = validateFile(file);
      if (error) {
        toast.error(error);
        setSelectedFile(null);
      } else {
        setSelectedFile(file);

        const fileInput = document.getElementById("file") as HTMLInputElement;
        if (fileInput) {
          const dataTransfer = new DataTransfer();
          dataTransfer.items.add(file);
          fileInput.files = dataTransfer.files;
        }
      }
    }
  };

  const handleRemoveFile = () => {
    setSelectedFile(null);
    const fileInput = document.getElementById("file") as HTMLInputElement;
    if (fileInput) {
      fileInput.value = "";
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    const formData = new FormData(e.currentTarget);
    const nama = formData.get("nama");
    if (!nama) {
      toast.error("Nama harus diisi");
      return;
    }

    if (!tanggal) {
      toast.error("Tanggal harus diisi");
      return;
    }

    if (!berkas && !selectedFile) {
      toast.error("File harus diunggah");
      return;
    }

    if (selectedFile) {
      const error = validateFile(selectedFile);
      if (error) {
        toast.error(error);
        return;
      }
    }

    setIsSubmitting(true);

    try {
      formData.set("tanggal", tanggal.toISOString());

      if (selectedFile) {
        formData.set("file", selectedFile);
      } else {
        formData.delete("file");
      }

      let result;
      if (berkas) {
        result = await updateBerkasPimpinan(berkas.id, formData);
      } else {
        result = await createBerkasPimpinan(formData);
      }

      if (result.error) {
        toast.error(result.error);
        setIsSubmitting(false);
      } else {
        toast.success(result.success);
        router.push("/dashboard/berkas-pimpinan");
        router.refresh();
      }
    } catch (error) {
      console.error("Error:", error);
      toast.error("Terjadi kesalahan saat menyimpan data");
      setIsSubmitting(false);
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
  };

  return (
    <form onSubmit={handleSubmit}>
      <Card>
        <CardContent className="p-6">
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label htmlFor="nama">
                  Nama <span className="text-red-500">*</span>
                </Label>
                <Input
                  id="nama"
                  name="nama"
                  defaultValue={berkas?.nama}
                  placeholder="Masukkan nama"
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="tanggal">
                  Tanggal <span className="text-red-500">*</span>
                </Label>
                <DatePicker
                  id="tanggal"
                  ariaLabel="Tanggal"
                  date={tanggal}
                  onDateChange={setTanggal}
                  placeholder="Pilih tanggal"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label htmlFor="catatan">Catatan</Label>
                <Textarea
                  id="catatan"
                  name="catatan"
                  defaultValue={berkas?.catatan || ""}
                  placeholder="Catatan tambahan (opsional)"
                  rows={4}
                  className="resize-none"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="file">
                  File Berkas <span className="text-red-500">*</span>
                </Label>
                <div
                  className={`border-2 border-dashed rounded-lg p-4 transition-colors flex flex-col justify-center ${
                    isDragging
                      ? userRole === "SEKRETARIS_CABANG"
                        ? "border-blue-500 bg-blue-50"
                        : "border-green-600 bg-emerald-50"
                      : "hover:border-slate-300"
                  }`}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                >
                  {!selectedFile && !berkas?.file ? (
                    <div className="text-center">
                      <Input
                        id="file"
                        name="file"
                        type="file"
                        onChange={handleFileChange}
                        accept=".pdf,.doc,.docx,.ppt,.pptx,.jpg,.jpeg,.png,.webp"
                        aria-required="true"
                        className="hidden"
                      />
                      <label
                        htmlFor="file"
                        className="cursor-pointer flex flex-col items-center gap-2"
                      >
                        <div className="p-3 bg-slate-100 rounded-full">
                          <Upload className="w-6 h-6 text-slate-500" />
                        </div>
                        <div className="text-sm">
                          <span
                            className={`font-medium ${
                              userRole === "SEKRETARIS_CABANG"
                                ? "text-blue-600"
                                : "text-green-600"
                            }`}
                          >
                            Klik untuk upload
                          </span>
                          <span className="text-muted-foreground ml-1">
                            atau drag & drop
                          </span>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          <AlertCircle className="w-3 h-3 inline mr-1" />
                          Maksimal 5MB. Format: PDF, Word, PPT, atau Gambar
                          (JPG/PNG/WebP).
                        </p>
                      </label>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-slate-100 rounded">
                          <FileText className="w-5 h-5 text-slate-500" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium truncate">
                            {selectedFile?.name || "File Lampiran (Tersimpan)"}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {selectedFile
                              ? formatFileSize(selectedFile.size)
                              : "File saat ini"}
                          </p>
                        </div>
                        <div className="flex items-center gap-1">
                          {!selectedFile && berkas?.file && (
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              asChild
                              className={`h-11 w-11 p-0 text-green-600 hover:text-emerald-700 hover:bg-emerald-50 ${
                                userRole === "SEKRETARIS_CABANG" &&
                                "text-blue-600 hover:text-blue-700 hover:bg-blue-50"
                              }`}
                              title="Buka file"
                            >
                              <a
                                href={`/api/berkas-pimpinan/download/${berkas.id}?preview=true${downloadToken ? `&token=${downloadToken}` : ""}`}
                                target="_blank"
                                aria-label="Buka file Berkas"
                              >
                                <Eye className="w-4 h-4" />
                              </a>
                            </Button>
                          )}
                          {selectedFile && (
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={handleRemoveFile}
                              className="h-11 w-11 p-0 text-red-500 hover:text-red-600 hover:bg-red-50"
                              title="Batalkan file pilihan"
                              aria-label="Batalkan file pilihan"
                            >
                              <X className="w-4 h-4" />
                            </Button>
                          )}
                        </div>
                      </div>

                      {!selectedFile && berkas?.file && (
                        <div className="flex items-center justify-between">
                          <Input
                            id="file"
                            name="file"
                            type="file"
                            onChange={handleFileChange}
                            accept=".pdf,.doc,.docx,.ppt,.pptx,.jpg,.jpeg,.png,.webp"
                            aria-required="true"
                            className="hidden"
                          />
                          <label
                            htmlFor="file"
                            className={`text-xs hover:underline cursor-pointer font-medium ${
                              userRole === "SEKRETARIS_CABANG"
                                ? "text-blue-600"
                                : "text-green-600"
                            }`}
                          >
                            Ganti file
                          </label>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>

            {isPdf(berkas?.file) && !selectedFile && (
              <div className="space-y-2 mt-4">
                <Label>Preview PDF</Label>
                <div className="border rounded-lg overflow-hidden bg-white shadow-sm">
                  <div className="p-2 border-b bg-slate-50/50 text-xs font-medium flex items-center justify-between px-3">
                    <div className="flex items-center gap-2">
                      <FileText className="w-3 h-3 text-primary" />
                      Pratinjau PDF
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-11 px-3 text-xs text-primary hover:text-primary hover:bg-primary/5"
                      asChild
                    >
                      <a
                        href={`/api/berkas-pimpinan/download/${berkas?.id}?preview=true${downloadToken ? `&token=${downloadToken}` : ""}`}
                        target="_blank"
                      >
                        <ExternalLink className="w-3 h-3 mr-1" />
                        Layar Penuh
                      </a>
                    </Button>
                  </div>
                  <div className="w-full min-h-[150px] md:h-[750px] bg-white relative">
                    {isMobile ? (
                      <div className="flex flex-col items-center justify-center p-8 text-center bg-slate-50/50 h-full">
                        <div className="p-3 bg-white rounded-full shadow-sm mb-3">
                          <Smartphone className="w-6 h-6 text-primary" />
                        </div>
                        <p className="text-xs font-medium text-slate-700 mb-1">
                          Pratinjau PDF di Mobile
                        </p>
                        <p className="text-[10px] text-slate-500 mb-4 max-w-[200px]">
                          Untuk kenyamanan terbaik, silakan buka PDF di layar
                          penuh
                        </p>
                        <Button
                          asChild
                          size="sm"
                          className="h-11 px-4 text-xs shadow-sm"
                        >
                          <a
                            href={`/api/berkas-pimpinan/download/${berkas?.id}?preview=true${downloadToken ? `&token=${downloadToken}` : ""}`}
                            target="_blank"
                          >
                            <ExternalLink className="w-3 h-3 mr-1.5" />
                            Buka PDF Sekarang
                          </a>
                        </Button>
                      </div>
                    ) : (
                      <>
                        {isPdfLoading && (
                          <div className="absolute inset-0 flex flex-col items-center justify-center bg-slate-50/50 z-10">
                            <Spinner className="h-8 w-8 mb-2 text-primary" />
                            <p className="text-[10px] text-muted-foreground animate-pulse">
                              Memuat...
                            </p>
                          </div>
                        )}
                        <object
                          data={`/api/berkas-pimpinan/download/${berkas?.id}?preview=true${downloadToken ? `&token=${downloadToken}` : ""}#view=FitH`}
                          type="application/pdf"
                          width="100%"
                          height="100%"
                          onLoad={() => setIsPdfLoading(false)}
                        >
                          <iframe
                            src={`/api/berkas-pimpinan/download/${berkas?.id}?preview=true${downloadToken ? `&token=${downloadToken}` : ""}`}
                            className="w-full h-full border-none"
                            title="PDF Preview"
                          />
                        </object>
                      </>
                    )}
                  </div>
                </div>
              </div>
            )}

            {isImage(berkas?.file) && !selectedFile && (
              <div className="space-y-2 mt-4">
                <Label>Preview Gambar</Label>
                <div className="border rounded-lg overflow-hidden bg-white shadow-sm">
                  <div className="p-2 border-b bg-slate-50/50 text-xs font-medium flex items-center justify-between px-3">
                    <div className="flex items-center gap-2">
                      <FileText className="w-3 h-3 text-primary" />
                      Pratinjau Gambar
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-11 px-3 text-xs text-primary hover:text-primary hover:bg-primary/5"
                      asChild
                    >
                      <a
                        href={`/api/berkas-pimpinan/download/${berkas?.id}?preview=true${downloadToken ? `&token=${downloadToken}` : ""}`}
                        target="_blank"
                      >
                        <ExternalLink className="w-3 h-3 mr-1" />
                        Layar Penuh
                      </a>
                    </Button>
                  </div>
                  <div className="w-full bg-slate-50 flex justify-center p-4 relative min-h-[200px]">
                    {isImageLoading && (
                      <div className="absolute inset-0 flex flex-col items-center justify-center bg-slate-50/50 z-10">
                        <Spinner className="h-8 w-8 mb-2 text-primary" />
                        <p className="text-[10px] text-muted-foreground animate-pulse">
                          Memuat...
                        </p>
                      </div>
                    )}
                    <img
                      src={
                        downloadToken
                          ? `/api/berkas-pimpinan/download/${berkas?.id}?preview=true&token=${downloadToken}`
                          : undefined
                      }
                      alt="Pratinjau Gambar"
                      className={`max-w-full max-h-[500px] rounded shadow-sm object-contain transition-opacity duration-300 ${isImageLoading ? "opacity-0" : "opacity-100"}`}
                      onLoad={() => setIsImageLoading(false)}
                    />
                  </div>
                </div>
              </div>
            )}
          </div>

          <div className="flex gap-3 mt-6 pt-6 border-t">
            <Button
              type="button"
              variant="outline"
              className="min-h-11 flex-1"
              onClick={() => router.push("/dashboard/berkas-pimpinan")}
              disabled={isSubmitting}
            >
              <X className="w-4 h-4 mr-2" />
              Batal
            </Button>
            <Button
              type="submit"
              disabled={isSubmitting}
              className={`min-h-11 flex-1 text-white shadow-md hover:shadow-xl transition-all duration-200 ${
                userRole === "SEKRETARIS_CABANG"
                  ? "bg-blue-600 hover:bg-blue-700"
                  : "bg-green-600 hover:bg-green-700"
              }`}
            >
              {isSubmitting ? (
                <>
                  <Spinner className="mr-2 h-4 w-4" />
                  Loading...
                </>
              ) : (
                <>
                  <Upload className="w-4 h-4 mr-2" />
                  Simpan Berkas
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </form>
  );
}
