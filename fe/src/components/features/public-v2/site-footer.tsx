"use client"

import Image from "next/image"
import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { Link2, Mail, MapPin, Phone } from "lucide-react"

import {
  scrollToHomeSection,
  type SectionId,
} from "@/components/features/public-v2/site-header"

export function SiteFooter() {
  const pathname = usePathname()
  const router = useRouter()

  function handleSectionLink(
    e: React.MouseEvent<HTMLAnchorElement>,
    id: SectionId
  ) {
    e.preventDefault()
    if (pathname === "/") {
      scrollToHomeSection(id)
    } else {
      sessionStorage.setItem("laci_scroll_target", id)
      router.push("/")
    }
  }

  return (
    <footer
      id="kontak"
      className="relative overflow-clip bg-[#041518] text-white"
    >
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 overflow-clip"
      >
        <div className="absolute -top-24 -right-24 h-80 w-80 rounded-full bg-[#0b5d33]/30 blur-3xl sm:h-[420px] sm:w-[480px]" />
        <div className="absolute -right-16 -bottom-20 h-72 w-72 rounded-full bg-[#0d7a42]/30 blur-3xl sm:h-96 sm:w-96" />
        <div className="absolute bottom-0 left-1/4 h-56 w-96 rounded-full bg-[#0a5c32]/20 blur-3xl" />

        <svg
          viewBox="0 0 1440 320"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          preserveAspectRatio="none"
          className="absolute inset-x-0 bottom-0 h-44 w-full sm:h-72"
        >
          <g stroke="#10b981" strokeWidth="1.2" fill="none">
            <path
              d="M-60 90 C 180 210, 380 270, 640 220 C 900 170, 1120 190, 1500 250"
              strokeOpacity="0.34"
            />
            <path
              d="M-60 110 C 185 224, 385 280, 648 232 C 910 182, 1125 202, 1500 262"
              strokeOpacity="0.28"
            />
            <path
              d="M-60 130 C 190 238, 390 290, 656 244 C 920 194, 1130 214, 1500 274"
              strokeOpacity="0.22"
            />
            <path
              d="M-60 150 C 195 252, 395 300, 664 256 C 930 206, 1135 226, 1500 286"
              strokeOpacity="0.17"
            />
            <path
              d="M-60 170 C 200 266, 400 310, 672 268 C 940 218, 1140 238, 1500 298"
              strokeOpacity="0.13"
            />
            <path
              d="M-60 190 C 205 280, 405 320, 680 280 C 950 230, 1145 250, 1500 310"
              strokeOpacity="0.09"
            />
          </g>
        </svg>
      </div>

      <div className="relative z-10 mx-auto max-w-6xl px-5 pt-11 pb-8 sm:px-6 sm:pt-12 lg:px-8">
        <div className="grid grid-cols-1 gap-8 md:grid-cols-2 lg:grid-cols-12 lg:gap-6">
          <div className="space-y-3.5 lg:col-span-4">
            <Link
              href="/"
              onClick={(e) => {
                if (pathname === "/") {
                  handleSectionLink(e, "home")
                }
              }}
              className="inline-flex items-center gap-3"
            >
              <Image
                src="/images/logo-laci.webp"
                alt="Logo Laci Digital"
                width={44}
                height={38}
                className="h-9 w-auto shrink-0 object-contain"
              />
              <div className="flex flex-col">
                <span className="text-[17px] font-extrabold tracking-tight text-white leading-none">
                  Laci Digital
                </span>
                <span className="mt-1 text-[11px] font-medium text-slate-400 leading-none">
                  PC IPNU IPPNU Magetan
                </span>
              </div>
            </Link>

            <p className="max-w-xs text-[13px] leading-relaxed text-slate-300/85 sm:text-xs">
              Laci Digital adalah platform administrasi digital untuk PC IPNU
              IPPNU Kabupaten Magetan, membantu pengelolaan administrasi
              organisasi yang lebih rapi, aman, dan efisien.
            </p>

            <div className="flex items-center gap-2.5 pt-1">
              <a
                href="https://pelajarnumagetan.or.id"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Website Pelajar NU Magetan"
                className="flex size-9 items-center justify-center rounded-full border border-white/12 bg-white/5 text-white/85 transition-colors hover:border-emerald-400/50 hover:bg-[#0a8744] hover:text-white sm:size-8"
              >
                <Link2 className="size-4 -rotate-45 sm:size-3.5" />
              </a>
            </div>
          </div>

          <div className="space-y-3 lg:col-span-1">
            <p className="text-[13px] font-bold tracking-wide text-white sm:text-xs">
              Menu
            </p>
            <ul className="space-y-2.5 text-[13px] text-slate-300/85 sm:text-xs">
              <li>
                <Link
                  href="/"
                  onClick={(e) => handleSectionLink(e, "home")}
                  className="transition-colors hover:text-white"
                >
                  Home
                </Link>
              </li>
              <li>
                <Link
                  href="/"
                  onClick={(e) => handleSectionLink(e, "fitur")}
                  className="transition-colors hover:text-white"
                >
                  Fitur
                </Link>
              </li>
              <li>
                <Link
                  href="/"
                  onClick={(e) => handleSectionLink(e, "resources")}
                  className="transition-colors hover:text-white"
                >
                  Resources
                </Link>
              </li>
              <li>
                <Link
                  href="/"
                  onClick={(e) => handleSectionLink(e, "kontak")}
                  className="transition-colors hover:text-white"
                >
                  Kontak
                </Link>
              </li>
            </ul>
          </div>

          <div className="space-y-3 lg:col-span-2 lg:pl-4">
            <p className="text-[13px] font-bold tracking-wide text-white sm:text-xs">
              Informasi
            </p>
            <ul className="space-y-2.5 text-[13px] text-slate-300/85 sm:text-xs">
              <li>
                <Link
                  href="/faq"
                  className={
                    pathname === "/faq"
                      ? "inline-flex items-center gap-1.5 font-semibold text-white"
                      : "transition-colors hover:text-white"
                  }
                >
                  {pathname === "/faq" && (
                    <span className="size-1.5 rounded-full bg-[#10b981]" />
                  )}
                  <span>FAQ</span>
                </Link>
              </li>
              <li>
                <Link
                  href="/ketentuan-penggunaan"
                  className={
                    pathname === "/ketentuan-penggunaan"
                      ? "inline-flex items-center gap-1.5 font-semibold text-white"
                      : "transition-colors hover:text-white"
                  }
                >
                  {pathname === "/ketentuan-penggunaan" && (
                    <span className="size-1.5 rounded-full bg-[#10b981]" />
                  )}
                  <span>Ketentuan Penggunaan</span>
                </Link>
              </li>
              <li>
                <Link
                  href="/kebijakan-privasi"
                  className={
                    pathname === "/kebijakan-privasi"
                      ? "inline-flex items-center gap-1.5 font-semibold text-white"
                      : "transition-colors hover:text-white"
                  }
                >
                  {pathname === "/kebijakan-privasi" && (
                    <span className="size-1.5 rounded-full bg-[#10b981]" />
                  )}
                  <span>Kebijakan Privasi</span>
                </Link>
              </li>
            </ul>
          </div>

          <div className="space-y-3 lg:col-span-3">
            <p className="text-[13px] font-bold tracking-wide text-white sm:text-xs">
              Kontak
            </p>
            <ul className="space-y-3 text-[13px] text-slate-300/90 sm:text-xs">
              <li className="flex items-center gap-2.5">
                <MapPin className="size-4 shrink-0 fill-white text-[#041518] sm:size-3.5" />
                <span>Magetan, Jawa Timur</span>
              </li>
              <li className="flex items-center gap-2.5">
                <Mail className="size-4 shrink-0 fill-white text-[#041518] sm:size-3.5" />
                <a
                  href="mailto:lacipelajarnumagetan@gmail.com"
                  className="break-all transition-colors hover:text-white sm:break-normal"
                >
                  lacipelajarnumagetan@gmail.com
                </a>
              </li>
              <li className="flex items-center gap-2.5">
                <Phone className="size-4 shrink-0 fill-white text-white sm:size-3.5" />
                <a
                  href="https://wa.me/6285850512135"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="transition-colors hover:text-white"
                >
                  +62 858-5051-2135
                </a>
              </li>
            </ul>
          </div>

          <div className="space-y-3 lg:col-span-2">
            <p className="text-[13px] font-bold tracking-wide text-white sm:text-xs">
              Media Partner
            </p>
            <a
              href="https://pelajarnumagetan.or.id"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex w-fit items-center justify-center rounded-xl bg-white px-3.5 py-2.5 shadow-md transition-transform hover:-translate-y-0.5"
            >
              <Image
                src="/images/media.webp"
                alt="Media Pelajar NU Magetan"
                width={145}
                height={50}
                className="h-10 w-auto object-contain sm:h-11"
              />
            </a>
          </div>
        </div>

        <div className="mt-9 border-t border-white/10 pt-5 text-xs text-slate-400/80 sm:mt-10">
          <p>© 2026 Laci Digital. Hak cipta dilindungi.</p>
        </div>
      </div>
    </footer>
  )
}

