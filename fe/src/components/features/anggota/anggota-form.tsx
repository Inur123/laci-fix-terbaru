"use client";

import React, { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { createAnggota, updateAnggota } from "@/app/actions/anggota-actions";
import {
  Upload,
  X,
  User,
  Phone,
  MapPin,
  Briefcase,
  CreditCard,
  Mail,
  Info,
  Plus,
  Trash2,
  GraduationCap,
  School,
} from "lucide-react";
import { Spinner } from "@/components/ui/spinner";
import { DatePicker } from "@/components/ui/date-picker";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ImageCropModal } from "@/components/shared/image-crop-modal";

interface AnggotaFormProps {
  anggota?: {
    id: string;
    foto?: string | null;
    updatedAt?: Date | string | null;
    tanggalLahir?: Date | string | null;
    namaLengkap?: string | null;
    nik?: string | null;
    nia?: string | null;
    jenisKelamin?: string | null;
    tempatLahir?: string | null;
    alamatLengkap?: string | null;
    noHp?: string | null;
    email?: string | null;
    jabatan?: string | null;
    noRfid?: string | null;
    pekerjaan?: string | null;
    hobi?: string | null;
    pendidikans?: Array<{
      jenjang: string;
      namaSekolah: string;
    }>;
    perkaderans?: Array<{
      namaPerkaderan: string;
      tanggal: Date | string;
      tempat: string;
    }>;
  };
  userRole?: string;
}

const toLocalYYYYMMDD = (dateInput: Date | string) => {
  const d = new Date(dateInput);
  if (isNaN(d.getTime())) return "";
  const offset = d.getTimezoneOffset();
  const localDate = new Date(d.getTime() - offset * 60 * 1000);
  return localDate.toISOString().split("T")[0];
};

export function AnggotaForm({ anggota, userRole }: AnggotaFormProps) {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const [previewImage, setPreviewImage] = useState<string | null>(
    anggota?.foto
      ? `/api/anggota/${anggota.id}/image?v=${anggota.updatedAt}`
      : null,
  );
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [removePhoto, setRemovePhoto] = useState(false);
  const [selectedDate, setSelectedDate] = useState<Date | undefined>(
    anggota?.tanggalLahir ? new Date(anggota.tanggalLahir) : undefined,
  );
  const [perkaderans, setPerkaderans] = useState<
    Array<{ namaPerkaderan: string; tanggal: string; tempat: string }>
  >(
    anggota?.perkaderans?.map((p) => ({
      namaPerkaderan: p.namaPerkaderan,
      tanggal: toLocalYYYYMMDD(p.tanggal),
      tempat: p.tempat,
    })) || [{ namaPerkaderan: "", tanggal: "", tempat: "" }],
  );

  const [pendidikans, setPendidikans] = useState<
    Array<{ jenjang: string; namaSekolah: string }>
  >(
    anggota?.pendidikans?.map((p) => ({
      jenjang: p.jenjang,
      namaSekolah: p.namaSekolah,
    })) || [{ jenjang: "", namaSekolah: "" }],
  );

  const [cropSrc, setCropSrc] = useState<string | null>(null);
  const [selectedFileMimeType, setSelectedFileMimeType] =
    useState<string>("image/jpeg");
  const [showCropModal, setShowCropModal] = useState(false);
  const [jenisKelamin, setJenisKelamin] = useState<string>(
    anggota?.jenisKelamin ?? "",
  );

  useEffect(() => {
    setPerkaderans((prev) =>
      prev.map((p) => {
        if (jenisKelamin === "LAKI_LAKI" && p.namaPerkaderan === "Latpel") {
          return { ...p, namaPerkaderan: "" };
        }
        if (jenisKelamin === "PEREMPUAN" && p.namaPerkaderan === "Latin") {
          return { ...p, namaPerkaderan: "" };
        }
        return p;
      }),
    );
  }, [jenisKelamin]);

  const addPerkaderan = () => {
    setPerkaderans([
      ...perkaderans,
      { namaPerkaderan: "", tanggal: "", tempat: "" },
    ]);
  };

  const removePerkaderan = (index: number) => {
    setPerkaderans(perkaderans.filter((_, i) => i !== index));
  };

  const updatePerkaderan = (index: number, field: string, value: string) => {
    const newPerkaderans = [...perkaderans];
    newPerkaderans[index] = { ...newPerkaderans[index], [field]: value };
    setPerkaderans(newPerkaderans);
  };

  const addPendidikan = () => {
    if (pendidikans.length >= 4) return;
    setPendidikans([...pendidikans, { jenjang: "", namaSekolah: "" }]);
  };

  const removePendidikan = (index: number) => {
    setPendidikans(pendidikans.filter((_, i) => i !== index));
  };

  const updatePendidikan = (index: number, field: string, value: string) => {
    const newPendidikans = [...pendidikans];
    newPendidikans[index] = { ...newPendidikans[index], [field]: value };
    setPendidikans(newPendidikans);
  };

  const handleFile = (file: File) => {
    if (!file.type.startsWith("image/")) {
      toast.error("File harus berupa gambar");
      return;
    }
    if (file.size > 2 * 1024 * 1024) {
      toast.error("Ukuran foto maksimal 2MB");
      return;
    }
    setSelectedFileMimeType(file.type);
    const reader = new FileReader();
    reader.onloadend = () => {
      setCropSrc(reader.result as string);
      setShowCropModal(true);
    };
    reader.readAsDataURL(file);
  };

  const handleCropComplete = (croppedFile: File) => {
    if (croppedFile.size > 2 * 1024 * 1024) {
      toast.error("Ukuran foto terlalu besar, coba lagi");
      return;
    }
    if (previewImage && previewImage.startsWith("blob:")) {
      URL.revokeObjectURL(previewImage);
    }
    setSelectedFile(croppedFile);
    setRemovePhoto(false);
    const objectUrl = URL.createObjectURL(croppedFile);
    setPreviewImage(objectUrl);
  };

  const handleImageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      handleFile(file);
      e.target.value = "";
    }
  };

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsSubmitting(true);

    const formData = new FormData(e.currentTarget);
    const formJenisKelamin = formData.get("jenisKelamin");

    if (!formJenisKelamin) {
      toast.error("Jenis Kelamin wajib diisi");
      setIsSubmitting(false);
      return;
    }

    const incompleteCadre = perkaderans.some((item) => {
      const hasValue = Boolean(
        item.namaPerkaderan.trim() || item.tanggal || item.tempat.trim(),
      );
      return (
        hasValue &&
        (!item.namaPerkaderan.trim() || !item.tanggal || !item.tempat.trim())
      );
    });
    if (incompleteCadre) {
      toast.error("Lengkapi nama, tanggal, dan tempat riwayat perkaderan");
      setIsSubmitting(false);
      return;
    }

    const incompleteEducation = pendidikans.some((item) => {
      const hasValue = Boolean(item.jenjang.trim() || item.namaSekolah.trim());
      return hasValue && (!item.jenjang.trim() || !item.namaSekolah.trim());
    });
    if (incompleteEducation) {
      toast.error("Lengkapi jenjang dan nama sekolah atau kampus");
      setIsSubmitting(false);
      return;
    }

    const activePerkaderans = perkaderans.filter(
      (p) => p.namaPerkaderan.trim() !== "",
    );

    const activePendidikans = pendidikans.filter(
      (p) => p.jenjang.trim() !== "",
    );

    if (selectedDate) formData.set("tanggalLahir", selectedDate.toISOString());
    if (selectedFile) {
      formData.set("foto", selectedFile);
      formData.delete("removeFoto");
    } else {
      formData.delete("foto");
      if (removePhoto) formData.set("removeFoto", "true");
    }
    formData.set("perkaderans", JSON.stringify(activePerkaderans));
    formData.set("pendidikans", JSON.stringify(activePendidikans));

    try {
      const result = anggota
        ? await updateAnggota(anggota.id, formData)
        : await createAnggota(formData);

      if (result.error) {
        toast.error(result.error);
        setIsSubmitting(false);
      } else {
        toast.success(result.success);
        router.push("/dashboard/anggota");
        router.refresh();
      }
    } catch {
      toast.error("Terjadi kesalahan sistem");
      setIsSubmitting(false);
    }
  };

  useEffect(() => {
    return () => {
      if (previewImage?.startsWith("blob:")) {
        URL.revokeObjectURL(previewImage);
      }
    };
  }, [previewImage]);

  return (
    <form onSubmit={handleSubmit} className="space-y-8">
      <div className="grid grid-cols-1 lg:grid-cols-[300px_1fr] lg:grid-rows-[auto_1fr] gap-x-8 gap-y-6">
        {/* SIDEBAR TOP: Foto & Alert */}
        <div className="lg:col-start-1 lg:row-start-1 space-y-6">
          <Card className="border-slate-200 shadow-sm overflow-hidden">
            <CardHeader className="bg-slate-50 border-b pb-4">
              <CardTitle className="text-sm font-bold flex items-center gap-2">
                <Upload
                  size={16}
                  className={
                    userRole === "SEKRETARIS_CABANG"
                      ? "text-blue-600"
                      : "text-green-600"
                  }
                />
                Foto Anggota
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-8 flex flex-col items-center">
              <div
                className={`relative group flex flex-col items-center justify-center w-40 h-40 rounded-full border-4 ${
                  isDragging
                    ? userRole === "SEKRETARIS_CABANG"
                      ? "border-blue-500 bg-blue-50"
                      : "border-green-600 bg-emerald-50"
                    : "border-white shadow-xl"
                } overflow-hidden transition-all duration-300 cursor-pointer`}
                onDragOver={(e) => {
                  e.preventDefault();
                  setIsDragging(true);
                }}
                onDragLeave={(e) => {
                  e.preventDefault();
                  setIsDragging(false);
                }}
                onDrop={(e) => {
                  e.preventDefault();
                  setIsDragging(false);
                  const file = e.dataTransfer.files?.[0];
                  if (file) handleFile(file);
                }}
              >
                <Avatar className="h-full w-full">
                  <AvatarImage
                    src={previewImage ?? undefined}
                    className="object-cover"
                  />
                  <AvatarFallback className="bg-slate-100 text-slate-400">
                    <User size={60} />
                  </AvatarFallback>
                </Avatar>

                <label
                  htmlFor="foto"
                  tabIndex={0}
                  role="button"
                  aria-label="Pilih foto anggota"
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      document.getElementById("foto")?.click();
                    }
                  }}
                  className={`absolute inset-0 flex flex-col items-center justify-center bg-black/50 text-white transition-opacity ${
                    isDragging || !previewImage
                      ? "opacity-100"
                      : "opacity-0 group-hover:opacity-100"
                  } cursor-pointer`}
                >
                  <Upload size={24} className="mb-1" />
                  <span className="text-[10px] font-medium">Upload Foto</span>
                </label>

                <input
                  type="file"
                  id="foto"
                  name="foto"
                  className="hidden"
                  accept="image/*"
                  onChange={handleImageChange}
                />
              </div>
              <p className="mt-4 text-xs text-slate-500 text-center">
                Drag & Drop atau Klik untuk upload.
                <br />
                Format: JPG, PNG, WEBP. Maks: 2MB.
              </p>
              {previewImage && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="mt-2 min-h-11 text-red-500 hover:text-red-600 hover:bg-red-50"
                  onClick={() => {
                    setPreviewImage(null);
                    setSelectedFile(null);
                    setRemovePhoto(Boolean(anggota?.foto));
                    const input = document.getElementById(
                      "foto",
                    ) as HTMLInputElement;
                    if (input) input.value = "";
                  }}
                >
                  <X size={14} className="mr-1" /> Hapus Foto
                </Button>
              )}
            </CardContent>
          </Card>

          <Alert
            className={
              userRole === "SEKRETARIS_CABANG"
                ? "bg-blue-50/50 border-blue-100"
                : "bg-emerald-50/50 border-emerald-100"
            }
          >
            <Info
              className={`h-4 w-4 ${userRole === "SEKRETARIS_CABANG" ? "text-blue-500" : "text-green-600"}`}
            />
            <AlertDescription
              className={`text-xs leading-relaxed font-medium ${userRole === "SEKRETARIS_CABANG" ? "text-blue-700" : "text-emerald-700"}`}
            >
              Data NIK, NIA, dan informasi sensitif lainnya akan dienkripsi
              otomatis sebelum disimpan.
            </AlertDescription>
          </Alert>
        </div>

        {/* MAIN CONTENT: Info Personal & Org */}
        <div className="lg:col-start-2 lg:row-start-1 lg:row-span-2 space-y-6">
          <Card className="border-slate-200 shadow-sm">
            <CardHeader className="bg-slate-50 border-b pb-4">
              <CardTitle className="text-sm font-bold flex items-center gap-2">
                <User
                  size={16}
                  className={
                    userRole === "SEKRETARIS_CABANG"
                      ? "text-blue-600"
                      : "text-green-600"
                  }
                />
                Informasi Personal
              </CardTitle>
              <CardDescription>
                Lengkapi data diri anggota sesuai dengan identitas resmi.
              </CardDescription>
            </CardHeader>
            <CardContent className="grid grid-cols-1 md:grid-cols-2 gap-6 pt-6">
              <div className="space-y-2">
                <Label htmlFor="namaLengkap" className="font-semibold">
                  Nama Lengkap <span className="text-red-500">*</span>
                </Label>
                <div className="relative">
                  <User
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <input
                    type="hidden"
                    name="perkaderans"
                    value={JSON.stringify(perkaderans)}
                  />
                  <Input
                    id="namaLengkap"
                    name="namaLengkap"
                    defaultValue={anggota?.namaLengkap ?? ""}
                    placeholder="Contoh: Irrandy"
                    className="pl-10"
                    maxLength={255}
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="email" className="font-semibold">
                  Email
                </Label>
                <div className="relative">
                  <Mail
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="email"
                    name="email"
                    type="email"
                    defaultValue={anggota?.email ?? ""}
                    placeholder="rekan@email.com"
                    className="pl-10"
                    maxLength={254}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="nik" className="font-semibold">
                  NIK (Nomor Induk Kependudukan)
                </Label>
                <div className="relative">
                  <CreditCard
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="nik"
                    name="nik"
                    defaultValue={anggota?.nik ?? ""}
                    placeholder="16 digit NIK"
                    className="pl-10"
                    inputMode="numeric"
                    pattern="[0-9]{16}"
                    minLength={16}
                    maxLength={16}
                    title="NIK harus terdiri dari 16 digit angka"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="nia" className="font-semibold">
                  NIA (Nomor Induk Anggota)
                </Label>
                <div className="relative">
                  <CreditCard
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="nia"
                    name="nia"
                    defaultValue={anggota?.nia ?? ""}
                    placeholder="Nomor Induk Anggota"
                    className="pl-10"
                    maxLength={100}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="jenisKelamin" className="font-semibold">
                  Jenis Kelamin <span className="text-red-500">*</span>
                </Label>
                <div className="relative">
                  <User
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 z-10"
                  />
                  <Select
                    name="jenisKelamin"
                    value={jenisKelamin}
                    onValueChange={setJenisKelamin}
                  >
                    <SelectTrigger id="jenisKelamin" className="w-full pl-10">
                      <SelectValue placeholder="Pilih Jenis Kelamin" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="LAKI_LAKI" className="text-xs">
                        Laki-laki
                      </SelectItem>
                      <SelectItem value="PEREMPUAN" className="text-xs">
                        Perempuan
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="noHp" className="font-semibold">
                  Nomor Handphone (WA)
                </Label>
                <div className="relative">
                  <Phone
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="noHp"
                    name="noHp"
                    defaultValue={anggota?.noHp ?? ""}
                    placeholder="Contoh: 085850512135"
                    className="pl-10"
                    inputMode="tel"
                    pattern="\+?[0-9]{8,15}"
                    maxLength={16}
                    title="Nomor handphone harus terdiri dari 8 sampai 15 digit angka"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="tempatLahir" className="font-semibold">
                  Tempat Lahir
                </Label>
                <div className="relative">
                  <MapPin
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="tempatLahir"
                    name="tempatLahir"
                    defaultValue={anggota?.tempatLahir ?? ""}
                    placeholder="Kota/Kabupaten"
                    className="pl-10"
                    maxLength={255}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="tanggalLahir" className="font-semibold">
                  Tanggal Lahir
                </Label>
                <DatePicker
                  id="tanggalLahir"
                  ariaLabel="Pilih tanggal lahir"
                  date={selectedDate}
                  onDateChange={setSelectedDate}
                  placeholder="Pilih tanggal lahir"
                />
              </div>

              <div className="space-y-2 md:col-span-2">
                <Label htmlFor="alamatLengkap" className="font-semibold">
                  Alamat Lengkap
                </Label>
                <div className="relative">
                  <MapPin
                    size={16}
                    className="absolute left-3 top-3 text-slate-400"
                  />
                  <Textarea
                    id="alamatLengkap"
                    name="alamatLengkap"
                    defaultValue={anggota?.alamatLengkap ?? ""}
                    placeholder="Masukkan alamat domisili lengkap"
                    rows={4}
                    className="pl-10 resize-none pt-2.5"
                    maxLength={2000}
                  />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="border-slate-200 shadow-sm">
            <CardHeader className="bg-slate-50 border-b pb-4">
              <CardTitle className="text-sm font-bold flex items-center gap-2">
                <Briefcase
                  size={16}
                  className={
                    userRole === "SEKRETARIS_CABANG"
                      ? "text-blue-600"
                      : "text-green-600"
                  }
                />
                Informasi Organisasi & Tambahan
              </CardTitle>
            </CardHeader>
            <CardContent className="grid grid-cols-1 md:grid-cols-2 gap-6 pt-6">
              <div className="space-y-2">
                <Label htmlFor="jabatan" className="font-semibold">
                  Jabatan
                </Label>
                <div className="relative">
                  <Briefcase
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="jabatan"
                    name="jabatan"
                    defaultValue={anggota?.jabatan ?? ""}
                    placeholder="Contoh: Ketua PAC"
                    className="pl-10"
                    maxLength={255}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="noRfid" className="font-semibold">
                  Nomor RFID / Kartu Anggota Digital
                </Label>
                <div className="relative">
                  <CreditCard
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="noRfid"
                    name="noRfid"
                    defaultValue={anggota?.noRfid ?? ""}
                    placeholder="ID Tag RFID (Jika ada)"
                    className="pl-10"
                    maxLength={100}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="pekerjaan" className="font-semibold">
                  Pekerjaan
                </Label>
                <div className="relative">
                  <Briefcase
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
                  />
                  <Input
                    id="pekerjaan"
                    name="pekerjaan"
                    defaultValue={anggota?.pekerjaan ?? ""}
                    placeholder="Contoh: Karyawan Swasta, Mahasiswa, dll"
                    className="pl-10"
                    maxLength={255}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="hobi" className="font-semibold">
                  Hobi / Minat Bakat
                </Label>
                <div className="relative">
                  <User
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 opacity-0"
                  />
                  <Input
                    id="hobi"
                    name="hobi"
                    defaultValue={anggota?.hobi ?? ""}
                    placeholder="Misal: Desain Grafis, Olahraga, dll"
                    maxLength={500}
                  />
                </div>
              </div>
            </CardContent>

            {/* Riwayat Pendidikan Inside Organisasi Card */}
            <div className="border-t border-slate-100 bg-slate-50/30">
              <div className="px-6 py-4 flex items-center justify-between">
                <h3 className="text-sm font-bold text-slate-700 flex items-center gap-2">
                  <School
                    size={16}
                    className={
                      userRole === "SEKRETARIS_CABANG"
                        ? "text-blue-600"
                        : "text-green-600"
                    }
                  />
                  Riwayat Pendidikan
                </h3>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addPendidikan}
                  disabled={pendidikans.length >= 4}
                  className={
                    userRole === "SEKRETARIS_CABANG"
                      ? "text-blue-600 border-blue-200 hover:bg-blue-50"
                      : "text-green-600 border-green-200 hover:bg-green-50"
                  }
                  aria-label="Tambah riwayat pendidikan"
                  title="Tambah riwayat pendidikan"
                >
                  <Plus size={14} className="mr-1" /> Tambah
                </Button>
              </div>

              <CardContent className="px-6 pb-6 pt-0">
                {pendidikans.length === 0 ? (
                  <div className="text-center py-6 border-2 border-dashed rounded-lg bg-white">
                    <p className="text-[10px] text-slate-500 font-medium">
                      Belum ada data pendidikan.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {pendidikans.map((p, index) => (
                      <div
                        key={index}
                        className="group relative grid grid-cols-1 md:grid-cols-2 gap-4 p-4 border rounded-lg bg-white hover:border-slate-300 transition-all shadow-sm pr-12"
                      >
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          onClick={() => removePendidikan(index)}
                          className="absolute top-1/2 -translate-y-1/2 right-1 min-h-11 min-w-11 text-slate-400 hover:text-red-600 hover:bg-red-50"
                          aria-label={`Hapus riwayat pendidikan ${index + 1}`}
                          title={`Hapus riwayat pendidikan ${index + 1}`}
                        >
                          <Trash2 size={16} />
                        </Button>

                        <div className="space-y-1.5">
                          <Label
                            htmlFor={`pendidikan-jenjang-${index}`}
                            className="text-[10px] font-bold uppercase text-slate-500"
                          >
                            Jenjang Pendidikan{" "}
                            <span className="text-red-500">*</span>
                          </Label>
                          <Select
                            value={p.jenjang}
                            onValueChange={(val) =>
                              updatePendidikan(index, "jenjang", val)
                            }
                          >
                            <SelectTrigger
                              id={`pendidikan-jenjang-${index}`}
                              className="min-h-11 w-full"
                            >
                              <SelectValue placeholder="Pilih Jenjang" />
                            </SelectTrigger>
                            <SelectContent>
                              {[
                                { id: "SD", label: "SD" },
                                { id: "MI", label: "MI" },
                                { id: "SMP", label: "SMP" },
                                { id: "MTs", label: "MTs" },
                                { id: "SMA", label: "SMA" },
                                { id: "SMK", label: "SMK" },
                                { id: "MAN", label: "MAN" },
                                { id: "KULIAH", label: "KULIAH" },
                              ].map((opt) => {
                                const isSelectedElsewhere = pendidikans.some(
                                  (item, i) =>
                                    i !== index &&
                                    item.jenjang?.toUpperCase() ===
                                      opt.id.toUpperCase(),
                                );
                                return (
                                  <SelectItem
                                    key={opt.id}
                                    value={opt.id}
                                    disabled={isSelectedElsewhere}
                                  >
                                    {opt.label}
                                  </SelectItem>
                                );
                              })}
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="space-y-1.5">
                          <Label
                            htmlFor={`pendidikan-sekolah-${index}`}
                            className="text-[10px] font-bold uppercase text-slate-500"
                          >
                            Nama Sekolah / Kampus{" "}
                            <span className="text-red-500">*</span>
                          </Label>
                          <Input
                            id={`pendidikan-sekolah-${index}`}
                            value={p.namaSekolah}
                            onChange={(e) =>
                              updatePendidikan(
                                index,
                                "namaSekolah",
                                e.target.value,
                              )
                            }
                            placeholder="Contoh: MAN 1 Magetan"
                            className="min-h-11"
                            maxLength={255}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </div>
          </Card>
        </div>

        {/* SIDEBAR BOTTOM / MOBILE BOTTOM: Perkaderan & Pendidikan */}
        <div className="lg:col-start-1 lg:row-start-2 space-y-6">
          <Card className="border-slate-200 shadow-sm">
            <CardHeader className="bg-slate-50 border-b pb-4">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-bold flex items-center gap-2">
                  <GraduationCap
                    size={16}
                    className={
                      userRole === "SEKRETARIS_CABANG"
                        ? "text-blue-600"
                        : "text-green-600"
                    }
                  />
                  Riwayat Perkaderan
                </CardTitle>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addPerkaderan}
                  className={
                    userRole === "SEKRETARIS_CABANG"
                      ? "text-blue-600 border-blue-200 hover:bg-blue-50"
                      : "text-green-600 border-green-200 hover:bg-green-50"
                  }
                  aria-label="Tambah riwayat perkaderan"
                  title="Tambah riwayat perkaderan"
                >
                  <Plus size={14} />
                </Button>
              </div>
            </CardHeader>
            <CardContent className="pt-4 px-3 pb-4">
              {perkaderans.length === 0 ? (
                <div className="text-center py-6 border-2 border-dashed rounded-lg bg-slate-50/50">
                  <p className="text-[10px] text-slate-500 font-medium">
                    Belum ada data.
                  </p>
                </div>
              ) : (
                <div className="space-y-4">
                  {perkaderans.map((p, index) => (
                    <div
                      key={index}
                      className="group relative space-y-3 p-3 border rounded-lg bg-white hover:border-slate-300 transition-all shadow-sm"
                    >
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        onClick={() => removePerkaderan(index)}
                        className="absolute top-1 right-1 min-h-11 min-w-11 text-slate-400 hover:text-red-600 hover:bg-red-50"
                        aria-label={`Hapus riwayat perkaderan ${index + 1}`}
                        title={`Hapus riwayat perkaderan ${index + 1}`}
                      >
                        <Trash2 size={12} />
                      </Button>

                      <div className="space-y-1">
                        <Label
                          htmlFor={`perkaderan-nama-${index}`}
                          className="text-[10px] font-bold uppercase text-slate-500"
                        >
                          Nama <span className="text-red-500">*</span>
                        </Label>
                        <Select
                          value={p.namaPerkaderan}
                          onValueChange={(val) =>
                            updatePerkaderan(index, "namaPerkaderan", val)
                          }
                        >
                          <SelectTrigger
                            id={`perkaderan-nama-${index}`}
                            className="min-h-11 text-xs w-full"
                          >
                            <SelectValue placeholder="Pilih Perkaderan" />
                          </SelectTrigger>
                          <SelectContent>
                            {[
                              { id: "Makesta", label: "Makesta" },
                              { id: "Lakmud", label: "Lakmud" },
                              { id: "Lakut", label: "Lakut" },
                              { id: "Diklatama", label: "Diklatama" },
                              { id: "Diklatmad", label: "Diklatmad" },
                              ...(jenisKelamin === "LAKI_LAKI"
                                ? [{ id: "Latin", label: "Latin" }]
                                : []),
                              ...(jenisKelamin === "PEREMPUAN"
                                ? [{ id: "Latpel", label: "Latpel" }]
                                : []),
                            ].map((opt) => {
                              const isSelectedElsewhere = perkaderans.some(
                                (item, i) =>
                                  i !== index &&
                                  item.namaPerkaderan?.toUpperCase() ===
                                    opt.id.toUpperCase(),
                              );
                              return (
                                <SelectItem
                                  key={opt.id}
                                  value={opt.id}
                                  disabled={isSelectedElsewhere}
                                  className="text-xs"
                                >
                                  {opt.label}
                                </SelectItem>
                              );
                            })}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-1">
                        <Label
                          htmlFor={`perkaderan-tanggal-${index}`}
                          className="text-[10px] font-bold uppercase text-slate-500"
                        >
                          Tanggal <span className="text-red-500">*</span>
                        </Label>
                        <DatePicker
                          id={`perkaderan-tanggal-${index}`}
                          ariaLabel={`Pilih tanggal perkaderan ${index + 1}`}
                          date={p.tanggal ? new Date(p.tanggal) : undefined}
                          onDateChange={(date) =>
                            updatePerkaderan(
                              index,
                              "tanggal",
                              date ? toLocalYYYYMMDD(date) : "",
                            )
                          }
                          placeholder="Pilih tanggal"
                        />
                      </div>
                      <div className="space-y-1">
                        <Label
                          htmlFor={`perkaderan-tempat-${index}`}
                          className="text-[10px] font-bold uppercase text-slate-500"
                        >
                          Tempat <span className="text-red-500">*</span>
                        </Label>
                        <Input
                          id={`perkaderan-tempat-${index}`}
                          value={p.tempat}
                          onChange={(e) =>
                            updatePerkaderan(index, "tempat", e.target.value)
                          }
                          placeholder="Contoh: PAC Magetan"
                          className="min-h-11 text-xs"
                          maxLength={255}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      <div className="flex gap-3 mt-6 pt-6 border-t">
        <Button
          type="button"
          variant="outline"
          className="flex-1 hover:bg-slate-100 transition-all duration-200"
          onClick={() =>
            router.push(
              anggota
                ? `/dashboard/anggota/${anggota.id}`
                : "/dashboard/anggota",
            )
          }
          disabled={isSubmitting}
        >
          <X className="w-4 h-4 mr-2" />
          Batal
        </Button>
        <Button
          type="submit"
          className={`flex-1 text-white shadow-md hover:shadow-xl transition-all duration-200 ${
            userRole === "SEKRETARIS_CABANG"
              ? "bg-blue-600 hover:bg-blue-700"
              : "bg-green-600 hover:bg-green-700"
          }`}
          disabled={isSubmitting}
        >
          {isSubmitting ? (
            <>
              <Spinner className="mr-2 h-4 w-4" />
              Loading...
            </>
          ) : (
            <>
              <Upload className="w-4 h-4 mr-2" />
              Simpan Data Anggota
            </>
          )}
        </Button>
      </div>

      {/* Crop Modal */}
      {cropSrc && (
        <ImageCropModal
          imageSrc={cropSrc}
          originalMimeType={selectedFileMimeType}
          open={showCropModal}
          onClose={() => {
            setShowCropModal(false);
            setCropSrc(null);
          }}
          onCropComplete={handleCropComplete}
        />
      )}
    </form>
  );
}
