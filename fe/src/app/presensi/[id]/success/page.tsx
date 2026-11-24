import {
  getPublicPresensiDetail,
  getParticipantDetail,
} from "@/app/actions/presensi-actions";
import { notFound } from "next/navigation";
import {
  Check,
  User,
  Mail,
  Phone,
  MapPin,
  Sparkles,
  Clock,
  Briefcase,
  Building2,
} from "lucide-react";
import { format } from "date-fns";
import { id as idLocale } from "date-fns/locale";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ConfettiEffect } from "@/components/shared/confetti-effect";
import { cn } from "@/lib/utils";

type SearchParams = Promise<{ id?: string; token?: string }>;

export default async function PresensiSuccessPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: SearchParams;
}) {
  const { id } = await params;
	const { id: participantId, token } = await searchParams;

  const data = await getPublicPresensiDetail(id);
  if (!data) notFound();

  const participant = participantId
    ? await getParticipantDetail(participantId, token)
    : null;

  const tanggalFormatted = format(new Date(data.tanggal), "dd MMMM yyyy", {
    locale: idLocale,
  });

  return (
    <div className="min-h-screen bg-slate-100 flex flex-col items-center py-0 sm:py-8 px-0 sm:px-4">
      <ConfettiEffect />
      {/* ── Main Column Wrapper (Full Mobile, Card Desktop) ── */}
      <div className="w-full max-w-md bg-white sm:rounded-[32px] shadow-2xl shadow-slate-200/50 overflow-hidden min-h-screen sm:min-h-0 flex flex-col">
        {/* Success Icon Area */}
        <div className="pt-10 pb-6 flex flex-col items-center text-center px-6 relative">
          {/* Sparkles hiasan */}
          <Sparkles className="absolute top-10 right-24 w-4 h-4 text-green-200" />
          <div className="absolute top-24 left-24 w-3 h-3 text-green-300 font-bold">
            +
          </div>

          <div className="relative mb-4">
            {/* Cloud/Blob Effect di belakang */}
            <div className="absolute inset-0 bg-green-100/50 blur-2xl rounded-full scale-150" />
            <div className="relative w-16 h-16 rounded-full border-2 border-green-500 flex items-center justify-center bg-white shadow-sm ring-8 ring-green-50">
              <Check className="w-9 h-9 text-green-600" strokeWidth={3} />
            </div>
          </div>

          <h1 className="text-xl font-bold text-green-800 tracking-tight">
            Presensi Sukses!
          </h1>
          <p className="text-green-600/70 text-[11px] font-bold uppercase tracking-[0.1em] mt-1">
            Data Anda berhasil dicatat
          </p>
        </div>

        {/* Info Content - Ultra Compact */}
        <div className="px-6 pb-8 pt-2 space-y-6 flex-1 flex flex-col">
          {/* Agenda Info Section */}
          <div className="space-y-3">
            <AgendaRow label="Nama Agenda" value={data.namaKegiatan} />
            <AgendaRow label="Tempat" value={data.tempat} />
            <AgendaRow label="Penyelenggara" value={data.penyelenggara} />

            <div className="grid grid-cols-2 gap-2">
              <AgendaRow label="Tanggal" value={tanggalFormatted} />
              <AgendaRow
                label="Waktu"
                value={`${data.jamMulai} – ${data.jamSelesai}`}
              />
            </div>
          </div>

          <div className="border-t border-slate-50"></div>

          {/* Participant Section */}
          <div className="space-y-4">
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">
              Data Terverifikasi:
            </p>

            {participant ? (
              <div className="space-y-3">
                {/* ID Profil Bar */}
                <div className="flex items-center gap-3 bg-slate-50/70 p-3 rounded-2xl border border-slate-100">
                  <div className="w-10 h-10 bg-green-100 rounded-full flex items-center justify-center text-green-700 font-bold shrink-0">
                    <User className="w-5 h-5" strokeWidth={2} />
                  </div>
                  <div className="min-w-0 flex-1">
                    <h2 className="text-sm font-bold text-slate-900 leading-tight truncate">
                      {participant.namaLengkap}
                    </h2>
                    <div className="flex items-center gap-1.5 mt-1 flex-wrap">
                      <span
                        className={cn(
                          "inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-bold border",
                          participant.organisasi === "IPNU"
                            ? "bg-emerald-50 text-emerald-700 border-emerald-200"
                            : participant.organisasi === "IPPNU"
                              ? "bg-rose-50 text-rose-700 border-rose-200"
                              : "bg-slate-100 text-slate-700 border-slate-200",
                        )}
                      >
                        {participant.organisasi === "UMUM"
                          ? "Eksternal"
                          : participant.organisasi}
                      </span>
                      {participant.organisasi !== "UMUM" &&
                        participant.tingkat && (
                          <span className="text-[11px] font-medium text-slate-500">
                            {participant.tingkat}
                          </span>
                        )}
                    </div>
                  </div>
                </div>

                {/* Detail List */}
                <div className="bg-slate-50/50 rounded-2xl border border-slate-100 p-3.5 space-y-2.5">
                  <DetailItem
                    icon={<Mail className="w-3.5 h-3.5" />}
                    label="Email"
                    value={participant.email}
                  />
                  <DetailItem
                    icon={<Phone className="w-3.5 h-3.5" />}
                    label="No. WhatsApp / HP"
                    value={participant.noHp}
                  />

                  {participant.organisasi === "UMUM" ? (
                    <div className="pt-2 border-t border-slate-100">
                      <DetailItem
                        icon={<Building2 className="w-3.5 h-3.5" />}
                        label="Instansi / Lembaga"
                        value={participant.instansi}
                      />
                    </div>
                  ) : (
                    <>
                      <div className="grid grid-cols-2 gap-2 pt-2 border-t border-slate-100">
                        <DetailItem
                          icon={<Building2 className="w-3.5 h-3.5" />}
                          label="Tingkat"
                          value={participant.tingkat}
                        />
                        <DetailItem
                          icon={<MapPin className="w-3.5 h-3.5" />}
                          label="Asal Pimpinan"
                          value={participant.asalPimpinan}
                        />
                      </div>
                      <DetailItem
                        icon={<Briefcase className="w-3.5 h-3.5" />}
                        label="Jabatan"
                        value={participant.jabatan}
                      />
                    </>
                  )}

                  <div className="pt-2 border-t border-slate-100">
                    <DetailItem
                      icon={<Clock className="w-3.5 h-3.5" />}
                      label="Waktu Presensi"
                      value={`Absen pada Jam ${format(new Date(participant.createdAt), "HH:mm", { locale: idLocale })} WIB`}
                    />
                  </div>
                </div>
              </div>
            ) : (
              <p className="text-slate-400 text-[10px] italic text-center">
                Memproses data...
              </p>
            )}
          </div>

          {/* Action Button - Lebih Kecil */}
          <div className="pt-2">
            <Button
              asChild
              className="w-full bg-green-600 hover:bg-green-700 text-white h-10 rounded-xl font-bold text-sm shadow-md shadow-green-100 transition-all"
            >
              <Link href={`/presensi/${id}`}>Selesai</Link>
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

function AgendaRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[9px] font-bold text-slate-400 uppercase tracking-tight">
        {label}
      </p>
      <p className="text-[13px] font-bold text-slate-700 leading-tight truncate">
        {value}
      </p>
    </div>
  );
}

function DetailItem({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value?: string | null;
}) {
  return (
    <div className="flex items-start gap-2.5 min-w-0">
      <div className="text-slate-400 shrink-0 mt-0.5">{icon}</div>
      <div className="min-w-0 flex-1">
        <p className="text-[9px] font-bold text-slate-400 uppercase tracking-wider leading-none mb-1">
          {label}
        </p>
        <p className="text-xs font-semibold text-slate-700 leading-tight break-words">
          {value || "-"}
        </p>
      </div>
    </div>
  );
}
