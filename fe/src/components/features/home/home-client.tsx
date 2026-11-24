"use client"

import * as React from "react"
import Image from "next/image"
import Link from "next/link"
import {
  ArrowRight,
  BarChart3,
  Bell,
  Calendar,
  Check,
  ChevronDown,
  ChevronRight,
  Download,
  FileCheck2,
  FileSpreadsheet,
  FileText,
  FolderOpen,
  Image as ImageIcon,
  LayoutDashboard,
  Mail,
  MessageSquare,
  Play,
  QrCode,
  Send,
  Settings,
  ShieldCheck,
  Sparkles,
  UserCheck,
  Users,
  Zap,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { getPublicStats } from "@/app/actions/dashboard-actions"
import { SiteFooter } from "@/components/features/public-v2/site-footer"
import {
  SiteHeader,
  scrollToHomeSection,
} from "@/components/features/public-v2/site-header"
import { Button } from "@/components/ui/button"

interface FeatureItem {
  title: string
  description: string
  icon: React.ComponentType<{ className?: string }>
  iconBg: string
  iconColor: string
}
const features: FeatureItem[] = [
  {
    title: "Periode Kepengurusan",
    description:
      "Switch cepat, filter otomatis, dan pengingat perubahan periode.",
    icon: Calendar,
    iconBg: "bg-[#e6f7ed]",
    iconColor: "text-[#0a8744]",
  },
  {
    title: "Data Anggota",
    description:
      "Kelola data anggota lengkap: foto, NIA, info pribadi, filter per periode.",
    icon: Users,
    iconBg: "bg-[#f3e8ff]",
    iconColor: "text-[#7c3aed]",
  },
  {
    title: "Arsip Berkas Digital",
    description:
      "Enkripsi, search, pagination, dan export Excel (format Indonesia).",
    icon: FolderOpen,
    iconBg: "bg-[#ffedd5]",
    iconColor: "text-[#ea580c]",
  },
  {
    title: "Arsip Surat Digital",
    description:
      "Surat masuk/keluar: filter jenis, search, dan pencarian nomor surat.",
    icon: Mail,
    iconBg: "bg-[#e0f2fe]",
    iconColor: "text-[#0284c7]",
  },
  {
    title: "Pengajuan Surat",
    description:
      "Status pending/diterima/ditolak dan notifikasi otomatis.",
    icon: Send,
    iconBg: "bg-[#f3e8ff]",
    iconColor: "text-[#7c3aed]",
  },
  {
    title: "Manajemen User PAC",
    description:
      "Approve akses, verifikasi email, dan monitoring aktivasi akun.",
    icon: UserCheck,
    iconBg: "bg-[#ffedd5]",
    iconColor: "text-[#ea580c]",
  },
  {
    title: "Kalender Kegiatan",
    description:
      "Jadwal cabang lengkap: waktu, tempat, dan deskripsi kegiatan.",
    icon: Calendar,
    iconBg: "bg-[#ffe4e6]",
    iconColor: "text-[#e11d48]",
  },
  {
    title: "Dashboard & Export",
    description:
      "Statistik real-time & export Excel format tanggal Indonesia.",
    icon: QrCode,
    iconBg: "bg-[#e6f7ed]",
    iconColor: "text-[#0a8744]",
  },
  {
    title: "Keamanan Data",
    description:
      "Role-based access, verifikasi email, dan autentikasi berlapis.",
    icon: ShieldCheck,
    iconBg: "bg-[#e0f2fe]",
    iconColor: "text-[#0284c7]",
  },
  {
    title: "Presensi Digital",
    description:
      "Sistem presensi QR Code real-time yang terintegrasi dengan agenda kegiatan.",
    icon: QrCode,
    iconBg: "bg-[#e6f7ed]",
    iconColor: "text-[#0a8744]",
  },
]

interface ResourceItem {
  title: string
  description: string
  href: string
  icon: React.ComponentType<{ className?: string }>
  iconBg: string
  iconColor: string
}

const resources: ResourceItem[] = [
  {
    title: "Format Surat Baru",
    description:
      "Template surat terbaru sesuai standar PC IPNU IPPNU Magetan.",
    href: "https://drive.google.com/drive/folders/1r-4OOy_5UcDDn6glvgz7NPrr6n5uxbSP",
    icon: FileText,
    iconBg: "bg-[#e8f2ff]",
    iconColor: "text-[#1d4ed8]",
  },
  {
    title: "Administrasi Makesta",
    description:
      "Format administrasi lengkap untuk Masa Kesetiaan Anggota (MAKESTA).",
    href: "https://drive.google.com/drive/folders/1e4__zQjlCTHsT_oIyrxlEeBDLUKm6KP7",
    icon: FileText,
    iconBg: "bg-[#e6f7ed]",
    iconColor: "text-[#0a8744]",
  },
  {
    title: "Logo Resmi",
    description:
      "Logo IPNU, IPPNU, dan PC Magetan berbagai format.",
    href: "https://drive.google.com/drive/folders/1cOwGh9FtPg62mDD61b7P097y-FWK-CTz",
    icon: ImageIcon,
    iconBg: "bg-[#e8f2ff]",
    iconColor: "text-[#1d4ed8]",
  },
  {
    title: "Peraturan IPNU IPPNU",
    description:
      "Peraturan organisasi, AD/ART, dan pedoman pelaksanaan terbaru.",
    href: "https://drive.google.com/drive/folders/1uJuqz-Y8CD5RT0cwVNn1V2Db6rI_3FHj",
    icon: FileCheck2,
    iconBg: "bg-[#ffe8e8]",
    iconColor: "text-[#dc2626]",
  },
  {
    title: "Format Pengajuan SP",
    description:
      "Template pengajuan Surat Pengantar dari PAC ke PC Magetan.",
    href: "https://drive.google.com/drive/folders/1l9Nb5O7hTyKVmuSgG8k13QmiHfBcLGRo",
    icon: FileText,
    iconBg: "bg-[#f3e8ff]",
    iconColor: "text-[#7c3aed]",
  },
  {
    title: "Perlengkapan Lakmud",
    description:
      "Daftar dan template perlengkapan untuk Latihan Kader Muda (LAKMUD).",
    href: "https://drive.google.com/drive/u/0/folders/1FtsEWTe32t-p1aZI0MV6M5djwfClh468",
    icon: FileSpreadsheet,
    iconBg: "bg-[#e8f2ff]",
    iconColor: "text-[#1d4ed8]",
  },
]

type Session = {
  user: unknown
  session?: unknown
  expires?: string
} | null

interface HomeClientProps {
  session: Session
  stats: {
    anggotaTerdaftar: number
    suratDiproses: number
  }
}

export default function HomeClient({ session, stats }: HomeClientProps) {
  const [currentStats, setCurrentStats] = React.useState(stats)
  const numberFormatter = React.useMemo(
    () => new Intl.NumberFormat("id-ID"),
    []
  )

  React.useEffect(() => {
    setCurrentStats(stats)
  }, [stats])

  React.useEffect(() => {
    const handleRealtime = async () => {
      const fresh = await getPublicStats()
      setCurrentStats({
        anggotaTerdaftar: fresh.anggotaCount,
        suratDiproses: fresh.suratCount,
      })
    }

    window.addEventListener("laci-realtime", handleRealtime)
    return () => window.removeEventListener("laci-realtime", handleRealtime)
  }, [])

  const isAuthenticated = Boolean(session?.user)

  return (
    <div className="flex min-h-screen flex-col overflow-x-clip bg-white">
      <div className="relative overflow-clip bg-gradient-to-br from-[#e3f6eb] via-[#f3fbf6] to-[#dff4e7]">
        <svg
          aria-hidden="true"
          viewBox="0 0 1440 760"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          preserveAspectRatio="none"
          className="pointer-events-none absolute inset-0 h-full w-full"
        >
          <path
            d="M-80 120 C 40 90, 120 180, 90 310 C 60 440, -40 480, -100 420 Z"
            fill="#d4f0df"
            opacity="0.55"
          />
          <path
            d="M980 -40 L 1480 -40 L 1480 680 L 1160 680 Z"
            fill="#d2eedd"
            opacity="0.55"
            className="hidden lg:block"
          />
          <path
            d="M-60 540 C 180 560, 420 690, 760 700 C 1060 710, 1280 640, 1480 610 L 1480 760 L -60 760 Z"
            fill="#d5f0e0"
            opacity="0.7"
          />
          <path
            d="M-40 690 C 260 745, 560 755, 920 730 C 1180 710, 1340 685, 1480 670 L 1480 765 L -40 765 Z"
            fill="#ffffff"
          />
        </svg>

        <SiteHeader isAuthenticated={isAuthenticated} />

        <section
          id="home"
          className="relative scroll-mt-24 pt-4 pb-12 sm:pt-10 sm:pb-24 lg:pb-32"
        >
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="grid grid-cols-1 items-center gap-8 sm:gap-10 lg:grid-cols-12 lg:gap-6">
              <div className="min-w-0 space-y-5 sm:space-y-6 lg:col-span-6">
                <div className="inline-flex items-center gap-2 rounded-full border border-[#bce8ce] bg-[#eefaf3] px-3.5 py-1.5 text-xs font-semibold text-slate-700 shadow-2xs sm:px-4">
                  <span className="size-2 rounded-full bg-[#0a8744]" />
                  <span>Layanan Cerdas Administrasi</span>
                </div>

                <h1 className="text-[2rem] leading-[1.14] font-extrabold tracking-tight text-[#091526] sm:text-5xl lg:text-[3.35rem] lg:leading-[1.12]">
                  Selamat Datang di{" "}
                  <span className="mt-0.5 block text-[#0a8744] sm:mt-1">
                    Laci Digital
                  </span>
                </h1>

                <p className="max-w-[520px] text-[13.5px] leading-relaxed text-slate-600 sm:text-[15px]">
                  <strong className="font-bold text-slate-800">
                    Laci Digital
                  </strong>{" "}
                  adalah platform administrasi untuk{" "}
                  <strong className="font-bold text-slate-800">
                    PC IPNU IPPNU Kabupaten Magetan.
                  </strong>{" "}
                  Kelola data anggota, arsip surat dan berkas, pengajuan surat,
                  serta administrasi organisasi dengan lebih mudah, aman, dan
                  efisien.
                </p>

                <div className="flex flex-wrap items-center gap-3 pt-0.5 sm:gap-4 sm:pt-1">
                  <Button
                    size="lg"
                    asChild
                    className="h-11 rounded-full bg-[#0a8744] px-5 text-xs font-bold text-white shadow-[0_10px_25px_-5px_rgba(10,135,68,0.45)] hover:bg-[#087038] sm:h-13 sm:px-8 sm:text-sm"
                  >
                    <Link href={isAuthenticated ? "/dashboard" : "/login"}>
                      <Play className="size-3.5 fill-current" />
                      {isAuthenticated ? "Dashboard" : "Masuk ke Sistem"}
                    </Link>
                  </Button>
                  <Button
                    variant="outline"
                    size="lg"
                    type="button"
                    onClick={() => scrollToHomeSection("fitur")}
                    className="h-11 cursor-pointer rounded-full border-[#64748b] bg-white px-5 text-xs font-bold text-[#091526] shadow-[0_8px_20px_-4px_rgba(0,0,0,0.05)] hover:bg-slate-50 sm:h-13 sm:px-8 sm:text-sm"
                  >
                    Lihat Fitur Sistem
                    <ArrowRight className="size-4 text-slate-700" />
                  </Button>
                </div>

                <div className="grid grid-cols-3 gap-2 pt-2 sm:max-w-lg sm:gap-6 sm:pt-4">
                  <div className="flex items-center gap-2 sm:gap-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-[#e0f5e9] text-[#0a8744] sm:size-11">
                      <ShieldCheck className="size-4 sm:size-5" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-xs font-extrabold text-[#091526] sm:text-[13px]">
                        Aman
                      </p>
                      <p className="text-[10.5px] leading-tight text-slate-500 sm:text-[11px]">
                        Data terenkripsi
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 sm:gap-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-[#e0f5e9] text-[#0a8744] sm:size-11">
                      <Zap className="size-4 sm:size-5" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-xs font-extrabold text-[#091526] sm:text-[13px]">
                        Mudah
                      </p>
                      <p className="text-[10.5px] leading-tight text-slate-500 sm:text-[11px]">
                        Kelola organisasi
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 sm:gap-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-[#e0f5e9] text-[#0a8744] sm:size-11">
                      <Users className="size-4 sm:size-5" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-xs font-extrabold text-[#091526] sm:text-[13px]">
                        24/7
                      </p>
                      <p className="text-[10.5px] leading-tight text-slate-500 sm:text-[11px]">
                        Akses online
                      </p>
                    </div>
                  </div>
                </div>
              </div>

              <div className="relative min-w-0 lg:col-span-6">
                <svg
                  aria-hidden="true"
                  viewBox="0 0 48 48"
                  className="pointer-events-none absolute -top-7 -right-4 hidden size-11 text-[#0a8744] sm:block"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="3.5"
                  strokeLinecap="round"
                >
                  <line x1="12" y1="20" x2="18" y2="6" />
                  <line x1="23" y1="26" x2="36" y2="13" />
                  <line x1="28" y1="36" x2="43" y2="31" />
                </svg>

                <div className="rounded-[24px] border-4 border-[#e8f6ee] bg-white p-2 shadow-[0_20px_50px_-15px_rgba(6,36,23,0.15)] transition-transform duration-500 sm:rounded-[28px] sm:border-[6px] sm:p-3 lg:[transform:perspective(1100px)_rotateY(-9deg)_rotateX(3deg)_rotateZ(1.5deg)] lg:hover:[transform:perspective(1100px)_rotateY(0deg)_rotateX(0deg)_rotateZ(0deg)]">
                  <div className="grid grid-cols-12 overflow-hidden rounded-2xl border border-slate-100 bg-[#fbfdfc]">
                    <div className="hidden flex-col justify-between border-r border-slate-100 bg-white p-3 sm:col-span-3 sm:flex">
                      <div className="space-y-3.5">
                        <div className="flex items-center gap-2 px-1">
                          <Image
                            src="/images/logo-laci.webp"
                            alt="Laci Digital"
                            width={23}
                            height={20}
                            className="h-4 w-auto object-contain"
                          />
                          <span className="text-[11px] font-extrabold text-[#091526] whitespace-nowrap">
                            Laci Digital
                          </span>
                        </div>

                        <div className="space-y-1">
                          <div className="flex items-center gap-2 rounded-xl bg-[#e3f6eb] px-2.5 py-2 text-[10.5px] font-bold text-[#0a8744]">
                            <span className="flex size-4 shrink-0 items-center justify-center rounded-md bg-[#0a8744] text-white">
                              <LayoutDashboard className="size-2.5" />
                            </span>
                            <span>Dashboard</span>
                          </div>
                          {[
                            { label: "Anggota", icon: Users },
                            { label: "Surat", icon: Mail },
                            { label: "Arsip Berkas", icon: FolderOpen },
                            { label: "Kegiatan", icon: Calendar },
                            { label: "Laporan", icon: FileSpreadsheet },
                            { label: "Pengaturan", icon: Settings },
                          ].map((m) => {
                            const MIcon = m.icon
                            return (
                              <div
                                key={m.label}
                                className="flex items-center gap-2 rounded-xl px-2.5 py-1.5 text-[10px] font-medium text-slate-500 whitespace-nowrap"
                              >
                                <MIcon className="size-3.5 shrink-0 text-slate-500" />
                                <span>{m.label}</span>
                              </div>
                            )
                          })}
                        </div>
                      </div>
                    </div>

                    <div className="col-span-12 min-w-0 space-y-2.5 p-3 sm:col-span-9 sm:p-4">
                      <div className="flex items-center justify-between rounded-xl bg-white px-2.5 py-1.5">
                        <div className="flex min-w-0 items-center gap-2">
                          <div className="flex size-7 shrink-0 items-center justify-center rounded-full bg-[#e3f6eb] text-[#0a8744]">
                            <ShieldCheck className="size-4" />
                          </div>
                          <div className="min-w-0">
                            <p className="truncate text-xs font-extrabold text-[#091526] leading-none">
                              PC IPNU IPPNU
                            </p>
                            <p className="mt-0.5 truncate text-[10px] text-slate-500 leading-none">
                              Kabupaten Magetan
                            </p>
                          </div>
                        </div>

                        <div className="flex shrink-0 items-center gap-2">
                          <Bell className="size-3.5 text-slate-500" />
                          <div className="flex items-center gap-1.5">
                            <span className="flex size-6 items-center justify-center rounded-full bg-[#0a8744] text-[9px] font-bold text-white">
                              AD
                            </span>
                            <span className="text-[10.5px] font-semibold text-slate-700">
                              Admin
                            </span>
                            <ChevronDown className="size-3 text-slate-500" />
                          </div>
                        </div>
                      </div>

                      {/* Statistik menggunakan data backend halaman publik. */}
                      <div className="grid grid-cols-2 gap-2 sm:gap-2.5">
                        <div className="rounded-2xl border border-slate-100 bg-white p-2.5 shadow-2xs sm:p-3">
                          <div className="flex items-start gap-2 sm:gap-2.5">
                            <div className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-[#eef4ff] text-[#2563eb] sm:size-9">
                              <Users className="size-4" />
                            </div>
                            <div className="min-w-0 flex-1">
                              <p className="text-[10px] leading-tight font-semibold text-slate-600">
                                Anggota Terdaftar
                              </p>
                              <div className="mt-1 flex flex-wrap items-center gap-1 sm:mt-0.5 sm:gap-1.5">
                                <span className="text-lg font-extrabold text-[#091526] sm:text-xl leading-none">
                                  {numberFormatter.format(
                                    currentStats.anggotaTerdaftar
                                  )}
                                </span>
                                <span className="inline-flex shrink-0 items-center rounded-full bg-[#e3f6eb] px-1.5 py-0.5 text-[9px] font-bold text-[#0a8744] whitespace-nowrap leading-none">
                                  Data langsung
                                </span>
                              </div>
                              <p className="mt-1 text-[9px] text-slate-500 leading-none">
                                Periode ini
                              </p>
                            </div>
                          </div>
                        </div>

                        <div className="rounded-2xl border border-slate-100 bg-white p-2.5 shadow-2xs sm:p-3">
                          <div className="flex items-start gap-2 sm:gap-2.5">
                            <div className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-[#fff3e6] text-[#ea580c] sm:size-9">
                              <FileText className="size-4" />
                            </div>
                            <div className="min-w-0 flex-1">
                              <p className="text-[10px] leading-tight font-semibold text-slate-600">
                                Surat Diproses
                              </p>
                              <div className="mt-1 flex flex-wrap items-center gap-1 sm:mt-0.5 sm:gap-1.5">
                                <span className="text-lg font-extrabold text-[#091526] sm:text-xl leading-none">
                                  {numberFormatter.format(
                                    currentStats.suratDiproses
                                  )}
                                </span>
                                <span className="inline-flex shrink-0 items-center rounded-full bg-[#e3f6eb] px-1.5 py-0.5 text-[9px] font-bold text-[#0a8744] whitespace-nowrap leading-none">
                                  Data langsung
                                </span>
                              </div>
                              <p className="mt-1 text-[9px] text-slate-500 leading-none">
                                Real-time
                              </p>
                            </div>
                          </div>
                        </div>
                      </div>

                      <div className="rounded-2xl border border-slate-100 bg-white p-2.5 shadow-2xs sm:p-3">
                        <p className="mb-2 text-[10.5px] font-extrabold text-[#091526]">
                          Status Layanan
                        </p>
                        <div className="flex flex-wrap items-center gap-1.5 sm:flex-nowrap">
                          <span className="inline-flex items-center gap-1.5 rounded-full bg-[#f4f7f9] px-2.5 py-1 text-[9.5px] font-semibold text-slate-700 whitespace-nowrap">
                            <span className="size-1.5 rounded-full bg-[#0a8744]" />
                            Arsip Digital
                          </span>
                          <span className="inline-flex items-center gap-1.5 rounded-full bg-[#f4f7f9] px-2.5 py-1 text-[9.5px] font-semibold text-slate-700 whitespace-nowrap">
                            <span className="size-1.5 rounded-full bg-[#0d9488]" />
                            Pengajuan Surat
                          </span>
                          <span className="inline-flex items-center gap-1.5 rounded-full bg-[#f4f7f9] px-2.5 py-1 text-[9.5px] font-semibold text-slate-700 whitespace-nowrap">
                            <span className="size-1.5 rounded-full bg-[#f59e0b]" />
                            Dashboard &amp; Export
                          </span>
                        </div>
                      </div>

                      {/* Ringkasan sengaja tidak menampilkan angka atau tren contoh. */}
                      <div className="rounded-2xl border border-slate-100 bg-white p-2.5 shadow-2xs sm:p-3">
                        <div className="flex min-h-20 items-center gap-3 rounded-xl bg-[#f8fbf9] px-3 py-2.5">
                          <div className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-[#e3f6eb] text-[#0a8744]">
                            <BarChart3 className="size-4" />
                          </div>
                          <div className="min-w-0">
                            <p className="text-[10.5px] font-extrabold text-[#091526]">
                              Data sistem terhubung
                            </p>
                            <p className="mt-1 text-[9.5px] leading-relaxed text-slate-500">
                              Ringkasan anggota dan surat diperbarui dari data
                              Laci Digital.
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="mt-3 flex items-center gap-3 rounded-2xl border border-slate-100 bg-white px-4 py-3 shadow-[0_12px_30px_rgba(6,36,23,0.08)] sm:absolute sm:-right-4 sm:-bottom-6 sm:mt-0 sm:max-w-[240px] sm:-rotate-2 sm:gap-3.5 sm:py-3.5 sm:shadow-[0_15px_35px_rgba(6,36,23,0.10)]">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-[#e3f6eb] text-[#0a8744] sm:size-10">
                    <BarChart3 className="size-4 sm:size-5" />
                  </div>
                  <p className="text-xs font-extrabold leading-snug text-[#091526]">
                    Administrasi lebih mudah, organisasi lebih maju.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <main className="flex-1">
        <section
          id="fitur"
          className="scroll-mt-20 bg-white py-10 sm:py-16 lg:py-20"
        >
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="mx-auto mb-8 max-w-2xl text-center sm:mb-12">
              <div className="mb-2.5 inline-flex items-center gap-1.5 rounded-full border border-[#bce8ce] bg-[#ebf8f0] px-4 py-1 text-xs font-bold text-[#0a8744] sm:mb-3">
                <Sparkles className="size-3.5" />
                <span>Fitur Unggulan</span>
              </div>
              <h2 className="text-2xl font-extrabold tracking-tight text-[#091526] sm:text-4xl">
                Fitur <span className="text-[#0a8744]">Unggulan</span>
              </h2>
              <p className="mt-2 text-xs leading-relaxed text-slate-500 sm:mt-2.5 sm:text-[15px]">
                Sistem terintegrasi untuk manajemen organisasi yang efisien,
                rapi, dan modern.
              </p>
            </div>

            <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2 sm:gap-5 lg:grid-cols-4">
              {features.map((item, index) => {
                const Icon = item.icon
                const startsCenteredLastPair =
                  features.length % 4 === 2 && index === features.length - 2
                return (
                  <div
                    key={item.title}
                    className={cn(
                      "group rounded-[20px] border border-slate-100 bg-white p-4 shadow-[0_4px_20px_rgba(0,0,0,0.03)] transition-all duration-200 hover:-translate-y-0.5 hover:border-emerald-200 hover:shadow-[0_12px_30px_rgba(10,135,68,0.07)] sm:rounded-[22px] sm:p-6",
                      startsCenteredLastPair && "lg:col-start-2"
                    )}
                  >
                    <div className="flex items-center justify-between">
                      <div
                        className={cn(
                          "flex size-10 items-center justify-center rounded-xl sm:size-12 sm:rounded-2xl",
                          item.iconBg,
                          item.iconColor
                        )}
                      >
                        <Icon className="size-5" />
                      </div>

                      <span className="flex size-7 items-center justify-center rounded-full border border-slate-200/70 bg-white text-slate-500 transition-colors group-hover:border-emerald-200 group-hover:bg-[#e6f7ed] group-hover:text-[#0a8744] sm:size-8">
                        <ChevronRight className="size-3.5 sm:size-4" />
                      </span>
                    </div>

                    <h3 className="mt-3.5 text-sm font-extrabold text-[#091526] sm:mt-5 sm:text-[15px]">
                      {item.title}
                    </h3>
                    <p className="mt-1 text-xs leading-relaxed text-slate-500 sm:mt-1.5">
                      {item.description}
                    </p>
                  </div>
                )
              })}
            </div>
          </div>
        </section>

        <section
          id="resources"
          className="scroll-mt-20 relative overflow-clip bg-gradient-to-b from-[#e9f8ef] via-[#f4fbf7] to-white pt-12 pb-14 sm:pt-20 sm:pb-24"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 1440 680"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            preserveAspectRatio="none"
            className="pointer-events-none absolute inset-0 h-full w-full"
          >
            <path
              d="M0 0 H1440 V45 C1220 15, 960 65, 620 45 C320 25, 140 10, 0 55 Z"
              fill="#ffffff"
            />
            <path
              d="M1180 60 C1290 120, 1390 220, 1460 320 L1460 180 C1380 90, 1290 30, 1180 60 Z"
              fill="#d5f0e0"
              opacity="0.65"
            />
          </svg>

          <div
            aria-hidden="true"
            className="pointer-events-none absolute top-4 -right-6 hidden size-20 rounded-full bg-[#ccecd9]/70 lg:block"
          />

          <div
            aria-hidden="true"
            className="pointer-events-none absolute top-14 left-4 h-14 w-20 bg-[radial-gradient(#86efac_1.75px,transparent_1.75px)] [background-size:12px_12px] opacity-55 sm:top-24 sm:left-8 sm:h-20 sm:w-28 sm:[background-size:14px_14px] sm:opacity-65"
          />

          <div className="relative mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="mx-auto mb-8 max-w-2xl text-center sm:mb-12">
              <div className="mb-2.5 inline-flex items-center gap-1.5 rounded-full border border-slate-200/80 bg-white px-4 py-1 text-xs font-bold text-slate-700 shadow-2xs sm:mb-3">
                <FolderOpen className="size-3.5 text-slate-600" />
                <span>Dokumen &amp; Resources</span>
              </div>
              <h2 className="text-2xl font-extrabold tracking-tight text-[#091526] sm:text-4xl">
                Pusat <span className="text-[#0a8744]">Sumber Daya</span>
              </h2>
              <p className="mt-2 text-xs leading-relaxed text-slate-500 sm:mt-2.5 sm:text-[15px]">
                Download template surat, format administrasi, peraturan, dan
                logo resmi IPNU IPPNU Magetan.
              </p>
            </div>

            <div className="grid grid-cols-1 gap-3.5 sm:gap-5 md:grid-cols-2 lg:grid-cols-3">
              {resources.map((item) => {
                const Icon = item.icon
                return (
                  <div
                    key={item.title}
                    className="group flex items-start gap-3.5 rounded-[20px] border border-slate-100 bg-white p-4 shadow-[0_4px_20px_rgba(0,0,0,0.03)] transition-all duration-200 hover:-translate-y-0.5 hover:border-emerald-200 hover:shadow-[0_12px_30px_rgba(10,135,68,0.07)] sm:gap-4 sm:rounded-[22px] sm:p-6"
                  >
                    <div
                      className={cn(
                        "flex size-11 shrink-0 items-center justify-center rounded-xl sm:size-12 sm:rounded-2xl",
                        item.iconBg,
                        item.iconColor
                      )}
                    >
                      <Icon className="size-5" />
                    </div>

                    <div className="min-w-0 flex-1">
                      <h3 className="text-sm font-extrabold text-[#091526] sm:text-[15px]">
                        {item.title}
                      </h3>
                      <p className="mt-1 text-xs leading-relaxed text-slate-500 sm:mt-1.5">
                        {item.description}
                      </p>
                      <a
                        href={item.href}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-2.5 inline-flex items-center gap-1.5 text-xs font-extrabold text-[#0a8744] transition-colors hover:text-[#087038] hover:underline sm:mt-3.5"
                      >
                        <span>Buka Dokumen</span>
                        <Download className="size-3.5" />
                      </a>
                    </div>
                  </div>
                )
              })}
            </div>

            <div className="relative mt-8 overflow-clip rounded-[22px] border border-[#bce8ce] bg-gradient-to-r from-[#d4f3e0] via-[#e9f9ef] to-[#ccf0db] p-5 shadow-[0_8px_30px_rgba(10,135,68,0.06)] sm:mt-10 sm:rounded-[26px] sm:px-10 sm:py-7">
              <div
                aria-hidden="true"
                className="pointer-events-none absolute -top-12 -left-10 size-48 rounded-full bg-[#bce8ce]/45 sm:size-56"
              />
              <div
                aria-hidden="true"
                className="pointer-events-none absolute -right-12 -bottom-16 size-52 rounded-full bg-[#b4e4c7]/50 sm:size-64"
              />
              <div
                aria-hidden="true"
                className="pointer-events-none absolute top-4 right-72 hidden h-20 w-24 bg-[radial-gradient(#10b981_1.5px,transparent_1.5px)] [background-size:12px_12px] opacity-25 lg:block"
              />

              <div className="relative flex flex-col items-start justify-between gap-5 sm:flex-row sm:items-center sm:gap-6">
                <div className="flex items-center gap-4 sm:gap-8">
                  <div className="relative flex h-16 w-20 shrink-0 items-center justify-center sm:h-20 sm:w-24">
                    <div className="absolute top-1 left-1 h-13 w-11 -rotate-12 rounded-xl bg-gradient-to-br from-[#16a34a] to-[#0a8744] shadow-md sm:h-16 sm:w-14">
                      <div className="-mt-1.5 ml-2 h-2 w-5 rounded-t-md bg-[#15803d] sm:h-2.5 sm:w-6" />
                    </div>
                    <div className="relative z-10 flex h-13 w-11 -rotate-3 flex-col justify-center gap-1 rounded-xl border border-slate-100 bg-white p-2 shadow-md sm:h-16 sm:w-13 sm:gap-1.5 sm:p-2.5">
                      <span className="h-1 w-3.5 rounded-full bg-[#16a34a] sm:h-1.5 sm:w-4" />
                      <span className="h-1 w-5 rounded-full bg-[#22c55e] sm:h-1.5 sm:w-7" />
                      <span className="mt-0.5 h-1 w-6 rounded-full bg-slate-200 sm:w-8" />
                      <span className="h-1 w-5 rounded-full bg-slate-200 sm:w-7" />
                      <span className="h-1 w-4 rounded-full bg-slate-200 sm:w-6" />
                    </div>
                    <div className="absolute right-0 bottom-0.5 z-20 flex size-7 rotate-6 items-center justify-center rounded-xl border-2 border-[#bbf7d0] bg-gradient-to-br from-[#22c55e] to-[#0a8744] text-white shadow-lg sm:bottom-1 sm:size-9">
                      <Check className="size-4 stroke-[3] sm:size-5" />
                    </div>
                  </div>

                  <div className="min-w-0">
                    <p className="text-[10px] font-extrabold tracking-[0.12em] text-[#0a8744] uppercase sm:text-[11px]">
                      Masih Ada Pertanyaan?
                    </p>
                    <h3 className="mt-0.5 text-lg font-extrabold text-[#091526] sm:mt-1 sm:text-2xl">
                      Butuh Bantuan?
                    </h3>
                    <p className="mt-0.5 text-xs leading-relaxed text-slate-600 sm:mt-1 sm:text-sm">
                      Hubungi tim kami jika ada pertanyaan seputar dokumen dan
                      resources.
                    </p>
                  </div>
                </div>

                <Button
                  size="lg"
                  asChild
                  className="h-11 w-full shrink-0 rounded-full bg-[#0a8744] px-6 text-xs font-bold text-white shadow-[0_10px_25px_-5px_rgba(10,135,68,0.45)] hover:bg-[#087038] sm:h-12 sm:w-auto sm:px-7 sm:text-sm"
                >
                  <a
                    href="https://wa.me/6285850512135"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <MessageSquare className="size-4" />
                    Hubungi Kami
                    <ArrowRight className="size-4" />
                  </a>
                </Button>
              </div>
            </div>
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  )
}
