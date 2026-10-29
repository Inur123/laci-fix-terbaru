"use client";

import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import {
  createKegiatan,
  type KegiatanItem,
  updateKegiatan,
} from "@/app/actions/agenda-kegiatan-actions";
import {
  Calendar as CalendarIcon,
  Upload,
  MapPin,
  Loader2,
  Clock,
  X,
  Type,
  AlignLeft,
  Palette,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { capitalizeName, cn } from "@/lib/utils";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { addDays } from "date-fns";
import { formatDate, formatTime, idLocale } from "@/lib/date-utils";
import { DateRange } from "react-day-picker";

const PRESET_COLORS = [
  { name: "Merah", value: "#ef4444", class: "bg-red-500 ring-red-500" },
  { name: "Oranye", value: "#f97316", class: "bg-orange-500 ring-orange-500" },
  { name: "Kuning", value: "#eab308", class: "bg-yellow-500 ring-yellow-500" },
  { name: "Hijau", value: "#16a34a", class: "bg-green-600 ring-green-600" },
  { name: "Biru", value: "#3b82f6", class: "bg-blue-500 ring-blue-500" },
  { name: "Ungu", value: "#a855f7", class: "bg-purple-500 ring-purple-500" },
  { name: "Pink", value: "#ec4899", class: "bg-pink-500 ring-pink-500" },
  { name: "Abu-abu", value: "#64748b", class: "bg-slate-500 ring-slate-500" },
];

interface KegiatanFormProps {
  kegiatan?: KegiatanItem;
  userRole?: string;
}

export function KegiatanForm({ kegiatan, userRole }: KegiatanFormProps) {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [dateOpen, setDateOpen] = useState(false);
  const [isSelectingEndDate, setIsSelectingEndDate] = useState(false);
  const [date, setDate] = useState<DateRange | undefined>(
    kegiatan
      ? {
          from: new Date(kegiatan.tanggalMulai),
          to: kegiatan.tanggalSelesai
            ? new Date(kegiatan.tanggalSelesai)
            : new Date(kegiatan.tanggalMulai),
        }
      : {
          from: new Date(),
          to: new Date(),
        },
  );

  const [selectedColor, setSelectedColor] = useState(
    kegiatan?.warna || PRESET_COLORS[4].value,
  );

  const startTime = kegiatan
    ? formatTime(new Date(kegiatan.tanggalMulai))
    : formatTime(new Date());
  const endTime = kegiatan?.tanggalSelesai
    ? formatTime(new Date(kegiatan.tanggalSelesai))
    : formatTime(addDays(new Date(), 0));

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setIsSubmitting(true);

    if (!date?.from) {
      toast.error("Silakan pilih tanggal kegiatan");
      setIsSubmitting(false);
      return;
    }

    const formData = new FormData(e.currentTarget);
    const submittedStartTime =
      formData.get("startTime")?.toString() || startTime;
    const submittedEndTime = formData.get("endTime")?.toString() || endTime;

    const startDateTime = new Date(date.from);
    const [startHour, startMinute] = submittedStartTime.split(":").map(Number);
    startDateTime.setHours(startHour, startMinute);

    const endDateTime = date.to ? new Date(date.to) : new Date(date.from);
    const [endHour, endMinute] = submittedEndTime.split(":").map(Number);
    endDateTime.setHours(endHour, endMinute);

    if (endDateTime < startDateTime) {
      toast.error("Waktu selesai tidak boleh lebih awal dari waktu mulai");
      setIsSubmitting(false);
      return;
    }

    formData.set("tanggalMulai", startDateTime.toISOString());
    if (date.to || endTime) {
      formData.set("tanggalSelesai", endDateTime.toISOString());
    }

    formData.set("warna", selectedColor);

    try {
      const result = kegiatan
        ? await updateKegiatan(kegiatan.id, formData)
        : await createKegiatan(formData);

      if (result.error) {
        toast.error(result.error);
        setIsSubmitting(false);
      } else {
        toast.success(result.success || "Berhasil disimpan");
        router.push("/dashboard/agenda-kegiatan");
        router.refresh();
      }
    } catch (error) {
      console.error(error);
      toast.error("Terjadi kesalahan sistem");
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      <Card>
        <CardContent className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="judul">
                  <Type size={16} className="text-slate-500 inline mr-2" />
                  Judul Kegiatan <span className="text-red-500">*</span>
                </Label>
                <Input
                  id="judul"
                  name="judul"
                  placeholder="Contoh: Rapat Pleno Gabungan..."
                  defaultValue={
                    kegiatan?.judul ? capitalizeName(kegiatan.judul) : ""
                  }
                  maxLength={255}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="lokasi">
                  <MapPin size={16} className="text-slate-500 inline mr-2" />
                  Lokasi
                </Label>
                <div className="relative">
                  <Input
                    id="lokasi"
                    name="lokasi"
                    placeholder="Contoh: Kantor PCNU Magetan"
                    defaultValue={
                      kegiatan?.lokasi ? capitalizeName(kegiatan.lokasi) : ""
                    }
                    maxLength={500}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="deskripsi">
                  <AlignLeft size={16} className="text-slate-500 inline mr-2" />
                  Deskripsi
                </Label>
                <Textarea
                  id="deskripsi"
                  name="deskripsi"
                  placeholder="Tambahkan detail kegiatan..."
                  defaultValue={
                    kegiatan?.deskripsi
                      ? capitalizeName(kegiatan.deskripsi)
                      : ""
                  }
                  className="min-h-[120px] resize-none"
                  rows={5}
                  maxLength={5000}
                />
              </div>
            </div>

            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="date">
                  Tanggal Pelaksanaan <span className="text-red-500">*</span>
                </Label>
                <Popover
                  open={dateOpen}
                  onOpenChange={(open) => {
                    setDateOpen(open);
                    if (open) setIsSelectingEndDate(false);
                  }}
                >
                  <PopoverTrigger asChild>
                    <Button
                      id="date"
                      aria-label="Pilih rentang tanggal kegiatan"
                      variant={"outline"}
                      className={cn(
                        "w-full justify-start text-left font-normal",
                        !date && "text-muted-foreground",
                      )}
                    >
                      <CalendarIcon className="mr-2 h-4 w-4" />
                      {date?.from ? (
                        date.to ? (
                          <>
                            {formatDate(date.from, "dd MMM yyyy")} -{" "}
                            {formatDate(date.to, "dd MMM yyyy")}
                          </>
                        ) : (
                          formatDate(date.from, "dd MMM yyyy")
                        )
                      ) : (
                        <span>Pilih tanggal</span>
                      )}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-auto p-0" align="start">
                    <Calendar
                      initialFocus
                      mode="range"
                      defaultMonth={date?.from}
                      selected={date}
                      onDayClick={(day) => {
                        if (!isSelectingEndDate || !date?.from) {
                          setDate({ from: day, to: undefined });
                          setIsSelectingEndDate(true);
                          return;
                        }

                        setDate(
                          day < date.from
                            ? { from: day, to: date.from }
                            : { from: date.from, to: day },
                        );
                        setIsSelectingEndDate(false);
                        setDateOpen(false);
                      }}
                      numberOfMonths={1}
                      captionLayout="dropdown"
                      locale={idLocale}
                      formatters={{
                        formatMonthDropdown: (date) => {
                          return date.toLocaleString("id-ID", {
                            month: "long",
                          });
                        },
                      }}
                    />
                  </PopoverContent>
                </Popover>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="startTime">Jam Mulai</Label>
                  <div className="relative">
                    <Clock className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                      id="startTime"
                      name="startTime"
                      type="time"
                      defaultValue={startTime}
                      className="min-h-11 pl-9"
                      required
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="endTime">
                    Jam Selesai <span className="text-red-500">*</span>
                  </Label>
                  <div className="relative">
                    <Clock className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                      id="endTime"
                      name="endTime"
                      type="time"
                      defaultValue={endTime}
                      className="min-h-11 pl-9"
                      required
                    />
                  </div>
                </div>
              </div>

              <div className="space-y-2">
                <Label>
                  <Palette size={16} className="text-slate-500 inline mr-2" />
                  Label Warna <span className="text-red-500">*</span>
                </Label>
                <div className="p-4 border rounded-lg bg-slate-50/50 space-y-3">
                  <div className="flex flex-wrap gap-3">
                    {PRESET_COLORS.map((color) => (
                      <button
                        key={color.value}
                        type="button"
                        onClick={() => setSelectedColor(color.value)}
                        className={cn(
                          "h-11 w-11 rounded-full border-2 transition-all cursor-pointer ring-offset-2",
                          color.class,
                          selectedColor === color.value
                            ? "border-slate-600 scale-110 shadow-md ring-2 ring-slate-400"
                            : "border-transparent hover:scale-105 hover:shadow-sm",
                        )}
                        aria-label={`Pilih warna ${color.name}`}
                        aria-pressed={selectedColor === color.value}
                        title={color.name}
                      />
                    ))}
                  </div>
                  <div className="text-xs font-medium text-muted-foreground flex items-center gap-2">
                    <span>Terpilih:</span>
                    <span
                      className="px-2 py-0.5 rounded text-white text-[10px] uppercase font-bold"
                      style={{ backgroundColor: selectedColor }}
                    >
                      {PRESET_COLORS.find((c) => c.value === selectedColor)
                        ?.name || "Custom"}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="flex gap-3 mt-6 pt-6 border-t">
            <Button
              type="button"
              variant="outline"
              className="min-h-11 flex-1"
              onClick={() => router.push("/dashboard/agenda-kegiatan")}
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
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Menyimpan...
                </>
              ) : (
                <>
                  <Upload className="mr-2 h-4 w-4" />
                  Simpan Kegiatan
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </form>
  );
}
