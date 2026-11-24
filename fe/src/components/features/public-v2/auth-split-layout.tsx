import * as React from "react"
import Image from "next/image"
import { Mail, ShieldCheck, Users, Zap } from "lucide-react"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"

export interface AuthFeatureItem {
  title: string
  description: string
  icon: React.ComponentType<{ className?: string }>
}

interface AuthSplitLayoutProps {
  topBadge?: string
  brandDescription?: string
  features?: AuthFeatureItem[]
  page?: "login" | "register" | "verify"
  cardTitle?: string
  cardDescription?: string
  alternateAction?: React.ReactNode
  children: React.ReactNode
}

export function AuthSplitLayout({
  topBadge,
  brandDescription,
  features,
  page,
  cardTitle,
  cardDescription,
  alternateAction,
  children,
}: AuthSplitLayoutProps) {
  const resolvedDescription =
    brandDescription ??
    (page === "login"
      ? "Sistem manajemen administrasi digital untuk pimpinan anak cabang IPNU dan IPPNU di Kabupaten Magetan."
      : "Bergabunglah bersama Laci Digital untuk pengelolaan administrasi yang lebih rapi, aman, dan efisien.")
  const resolvedFeatures =
    features ??
    (page === "login"
      ? [
          {
            title: "Aman",
            description: "Data terenkripsi dan terlindungi",
            icon: ShieldCheck,
          },
          {
            title: "Mudah",
            description: "Pengelolaan organisasi lebih efisien",
            icon: Zap,
          },
          {
            title: "Terintegrasi",
            description: "Akses kapan saja dan di mana saja",
            icon: Users,
          },
        ]
      : [
          {
            title: "Pendaftaran mudah",
            description: "Hanya beberapa langkah",
            icon: ShieldCheck,
          },
          {
            title: "Verifikasi email",
            description: "Keamanan akun lebih terjamin",
            icon: Mail,
          },
          {
            title: "Akses lengkap",
            description: "Setelah akun aktif dan disetujui",
            icon: Users,
          },
        ])
  const resolvedTopBadge =
    topBadge ?? (page === "login" ? "Layanan Cerdas Administrasi" : undefined)
  const formBadge =
    page === "login"
      ? "Masuk ke Laci Digital"
      : page === "verify"
        ? "Verifikasi akun"
        : page === "register"
          ? "Daftar ke Laci Digital"
          : undefined

  return (
    <div className="relative flex min-h-screen flex-col justify-center overflow-x-clip bg-gradient-to-br from-[#dcefe4] via-[#ecf8f1] to-[#d4ebde]">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 overflow-clip"
      >
        <div className="absolute -top-20 -left-24 size-80 rounded-full bg-[#c7e9d5]/55 blur-2xl" />
        <div className="absolute top-1/3 -right-24 size-96 rounded-full bg-[#c2e6d1]/50 blur-2xl" />
        <div className="absolute -bottom-24 left-1/4 size-80 rounded-full bg-[#c9ead7]/45 blur-2xl" />
        <div className="absolute bottom-8 left-4 h-20 w-24 bg-[radial-gradient(#86efac_1.75px,transparent_1.75px)] [background-size:14px_14px] opacity-55" />
        <div className="absolute right-4 bottom-24 h-20 w-24 bg-[radial-gradient(#86efac_1.75px,transparent_1.75px)] [background-size:14px_14px] opacity-55" />
      </div>

      <main className="relative z-10 flex flex-1 items-center justify-center px-4 py-6 sm:px-6 sm:py-10">
        <div className="w-full max-w-[980px]">
          <Card className="grid gap-0 overflow-hidden rounded-[26px] border border-white/90 bg-white py-0 shadow-[0_20px_60px_-15px_rgba(6,36,23,0.12)] sm:rounded-[30px] md:grid-cols-12">
            <div
              className={cn(
                "relative hidden flex-col justify-start overflow-clip bg-gradient-to-br from-[#086036] via-[#044024] to-[#022414] px-9 pb-10 text-white md:col-span-6 md:flex lg:px-11 lg:pb-12",
                resolvedTopBadge ? "pt-10 lg:pt-11" : "pt-12 lg:pt-14"
              )}
            >
              <div
                aria-hidden="true"
                className="pointer-events-none absolute inset-0 overflow-clip"
              >
                <div className="absolute -top-20 -left-20 size-72 rounded-full bg-[#10b981]/25 blur-3xl" />
                <div className="absolute -right-16 -bottom-16 size-72 rounded-full bg-[#0d9450]/30 blur-2xl" />

                <div className="absolute top-7 right-7 h-24 w-28 bg-[radial-gradient(#34d399_1.5px,transparent_1.5px)] [background-size:15px_15px] opacity-30" />
                <div className="absolute bottom-7 left-7 h-24 w-28 bg-[radial-gradient(#34d399_1.5px,transparent_1.5px)] [background-size:15px_15px] opacity-30" />

                <svg
                  viewBox="0 0 500 680"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                  preserveAspectRatio="none"
                  className="absolute inset-0 h-full w-full"
                >
                  <path
                    d="M360 -20 C 440 120, 480 280, 400 440 C 330 580, 410 640, 520 690 L 520 -20 Z"
                    fill="#0b7240"
                    fillOpacity="0.32"
                  />
                  <path
                    d="M270 540 C 330 480, 430 490, 510 560 L 510 690 L 230 690 C 235 620, 245 565, 270 540 Z"
                    fill="#0e844a"
                    fillOpacity="0.28"
                  />
                  <path
                    d="M-20 530 C 90 560, 190 610, 310 690"
                    stroke="#34d399"
                    strokeOpacity="0.16"
                    strokeWidth="1.25"
                  />
                  <path
                    d="M-20 555 C 85 585, 180 630, 290 695"
                    stroke="#34d399"
                    strokeOpacity="0.11"
                    strokeWidth="1.25"
                  />
                </svg>
              </div>

              <div className="relative z-10 space-y-7">
                {resolvedTopBadge && (
                  <Badge
                    variant="outline"
                    className="gap-2 rounded-full border-emerald-400/30 bg-emerald-950/45 px-3.5 py-1.5 text-xs font-semibold text-emerald-100 backdrop-blur-xs"
                  >
                    <ShieldCheck className="size-3.5 text-emerald-300" />
                    <span>{resolvedTopBadge}</span>
                  </Badge>
                )}

                <div className="flex items-center gap-3.5">
                  <Image
                    src="/images/logo-laci.webp"
                    alt="Logo Laci Digital PC IPNU IPPNU Magetan"
                    width={56}
                    height={48}
                    priority
                    className="h-12 w-auto shrink-0 object-contain drop-shadow-sm"
                  />
                  <div>
                    <h2 className="text-2xl font-extrabold tracking-tight text-white leading-none">
                      Laci Digital
                    </h2>
                    <p className="mt-1.5 text-xs font-medium text-emerald-100/85 leading-none">
                      PC IPNU IPPNU Magetan
                    </p>
                  </div>
                </div>

                <p className="max-w-sm text-[14.5px] leading-relaxed text-emerald-50/90">
                  {resolvedDescription}
                </p>

                <div className="space-y-4 pt-1">
                  {resolvedFeatures.map((item) => {
                    const Icon = item.icon
                    return (
                      <div
                        key={item.title}
                        className="flex items-center gap-3.5"
                      >
                        <div className="flex size-11 shrink-0 items-center justify-center rounded-full border border-emerald-300/25 bg-[#0b7240]/65 text-emerald-200 shadow-inner">
                          <Icon className="size-5" />
                        </div>
                        <div className="min-w-0">
                          <p className="text-sm font-extrabold text-white leading-snug">
                            {item.title}
                          </p>
                          <p className="mt-0.5 text-xs text-emerald-100/80 leading-relaxed">
                            {item.description}
                          </p>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>

            <div className="relative flex flex-col justify-center bg-gradient-to-b from-[#effaf4] via-white to-white px-5 py-7 sm:px-9 sm:py-10 md:col-span-6 md:bg-none md:bg-white lg:px-11 lg:py-11">
              <div
                aria-hidden="true"
                className="pointer-events-none absolute top-3 left-3 h-12 w-16 bg-[radial-gradient(#86efac_1.5px,transparent_1.5px)] [background-size:12px_12px] opacity-35 md:hidden"
              />

              <div className="relative z-10 mx-auto w-full max-w-[400px]">
                {cardTitle && (
                  <div className="mb-6 text-center md:text-left">
                    {formBadge && (
                      <Badge
                        variant="secondary"
                        className="mb-3 rounded-full border-transparent bg-[#e1f5ea] px-3.5 py-1 text-xs font-bold text-[#0a8744]"
                      >
                        {formBadge}
                      </Badge>
                    )}
                    <h1 className="text-2xl font-extrabold tracking-tight text-[#091526] sm:text-[28px]">
                      {cardTitle}
                    </h1>
                    {cardDescription && (
                      <p className="mt-1.5 text-xs leading-relaxed text-slate-500 sm:text-[13.5px]">
                        {cardDescription}
                      </p>
                    )}
                  </div>
                )}
                {children}
                {alternateAction && (
                  <div className="mt-6 text-center text-sm text-slate-600">
                    {alternateAction}
                  </div>
                )}
              </div>
            </div>
          </Card>
        </div>
      </main>
    </div>
  )
}

