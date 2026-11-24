"use client"

import * as React from "react"
import Image from "next/image"
import Link from "next/link"
import { usePathname, useRouter } from "next/navigation"
import { ArrowRight, Menu } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"

export type SectionId = "home" | "fitur" | "resources" | "kontak"

interface SectionNavItem {
  id: SectionId
  label: string
}

const sectionNavItems: SectionNavItem[] = [
  { id: "home", label: "Home" },
  { id: "fitur", label: "Fitur" },
  { id: "resources", label: "Resources" },
  { id: "kontak", label: "Kontak" },
]

export function scrollToHomeSection(id: SectionId) {
  if (typeof window === "undefined") return
  document.documentElement.setAttribute("data-section", id)
  try {
    sessionStorage.setItem("laci_nav_section", id)
    sessionStorage.setItem("laci_nav_scrolled", id === "home" ? "0" : "1")
  } catch {}
  window.dispatchEvent(new CustomEvent("laci:nav-select", { detail: id }))
  if (id === "home") {
    window.scrollTo({ top: 0, behavior: "smooth" })
  } else {
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "start" })
    }
  }
  if (window.location.hash) {
    window.history.replaceState(null, "", window.location.pathname)
  }
}

export function BrandMark({ className }: { className?: string }) {
  return (
    <div className={cn("flex items-center gap-2.5 sm:gap-3", className)}>
      <Image
        src="/images/logo-laci.webp"
        alt="Logo Laci Digital PC IPNU IPPNU Magetan"
        width={44}
        height={38}
        priority
        className="h-8 w-auto shrink-0 object-contain sm:h-9"
      />
      <div className="flex flex-col">
        <span className="text-[14px] font-extrabold tracking-tight text-[#091526] sm:text-[15px] leading-none">
          Laci Digital
        </span>
        <span className="mt-1 text-[10px] font-medium text-slate-500 sm:text-[10.5px] leading-none">
          PC IPNU IPPNU Magetan
        </span>
      </div>
    </div>
  )
}

const useIsomorphicLayoutEffect =
  typeof window !== "undefined" ? React.useLayoutEffect : React.useEffect

interface SiteHeaderProps {
  isAuthenticated?: boolean
}

export function SiteHeader({ isAuthenticated = false }: SiteHeaderProps) {
  const pathname = usePathname()
  const router = useRouter()
  const [isMounted, setIsMounted] = React.useState(false)
  const [activeSection, setActiveSection] = React.useState<SectionId>("home")
  const [isScrolled, setIsScrolled] = React.useState(false)
  const [mobileOpen, setMobileOpen] = React.useState(false)

  React.useEffect(() => {
    setIsMounted(true)
  }, [])

  const isHomePath = isMounted && pathname === "/"

  React.useEffect(() => {
    setMobileOpen(false)
  }, [pathname])

  React.useEffect(() => {
    if (typeof window === "undefined") return
    const mql = window.matchMedia("(min-width: 768px)")
    const handleMediaChange = (e: MediaQueryListEvent) => {
      if (e.matches) {
        setMobileOpen(false)
      }
    }
    mql.addEventListener("change", handleMediaChange)
    return () => mql.removeEventListener("change", handleMediaChange)
  }, [])

  React.useEffect(() => {
    if (pathname !== "/") return
    const pending = sessionStorage.getItem(
      "laci_scroll_target"
    ) as SectionId | null
    if (pending) {
      sessionStorage.removeItem("laci_scroll_target")
      setTimeout(() => {
        scrollToHomeSection(pending)
        setActiveSection(pending)
      }, 60)
    }
  }, [pathname])

  // Sync header state before the hydration paint.
  useIsomorphicLayoutEffect(() => {
    if (typeof window === "undefined") return

    const navEntry = performance.getEntriesByType?.("navigation")?.[0] as
      | PerformanceNavigationTiming
      | undefined
    const isPageReload = navEntry?.type === "reload"
    let isRestorationComplete = !isPageReload
    let programmaticLockUntil = 0

    const applyActiveSection = (sec: SectionId) => {
      setActiveSection(sec)
      document.documentElement.setAttribute("data-section", sec)
      sessionStorage.setItem("laci_nav_section", sec)
    }

    const updateFromScroll = (fromUserScrollEvent: boolean) => {
      const scrollY =
        window.scrollY ||
        window.pageYOffset ||
        document.documentElement.scrollTop ||
        0
      const storedScrolled = sessionStorage.getItem("laci_nav_scrolled") === "1"

      // Preserve stored scroll state until reload restoration completes.
      const nextScrolled =
        isRestorationComplete || fromUserScrollEvent
          ? scrollY > 24
          : scrollY > 24 || storedScrolled

      setIsScrolled(nextScrolled)
      document.documentElement.setAttribute(
        "data-scrolled",
        nextScrolled ? "true" : "false"
      )

      if (isRestorationComplete || fromUserScrollEvent || scrollY > 24) {
        sessionStorage.setItem("laci_nav_scrolled", nextScrolled ? "1" : "0")
      }

      if (pathname !== "/") return
      if (Date.now() < programmaticLockUntil) return

      if (scrollY < 200) {
        if (isRestorationComplete || fromUserScrollEvent) {
          applyActiveSection("home")
        }
        return
      }

      const reachedBottom =
        window.innerHeight + scrollY >=
        document.documentElement.scrollHeight - 80
      if (reachedBottom) {
        applyActiveSection("kontak")
        return
      }

      const ids: SectionId[] = ["fitur", "resources", "kontak"]
      let current: SectionId = "home"
      for (const id of ids) {
        const el = document.getElementById(id)
        if (el) {
          const rect = el.getBoundingClientRect()
          if (rect.top <= 240) {
            current = id
          }
        }
      }
      applyActiveSection(current)
    }

    const savedSec = sessionStorage.getItem(
      "laci_nav_section"
    ) as SectionId | null
    if (isPageReload && savedSec && pathname === "/") {
      setActiveSection(savedSec)
      document.documentElement.setAttribute("data-section", savedSec)
    } else {
      setActiveSection("home")
      document.documentElement.setAttribute("data-section", "home")
    }

    updateFromScroll(false)

    // Delay transitions until browser scroll restoration settles.
    const settleTimer = window.setTimeout(() => {
      isRestorationComplete = true
      updateFromScroll(true)
      document.documentElement.setAttribute("data-nav-ready", "true")
    }, 350)

    const handleScroll = () => {
      updateFromScroll(true)
    }

    const handleNavSelect = (e: Event) => {
      const custom = e as CustomEvent<SectionId>
      if (custom.detail) {
        programmaticLockUntil = Date.now() + 650
        applyActiveSection(custom.detail)
      }
    }

    window.addEventListener("scroll", handleScroll, { passive: true })
    window.addEventListener("laci:nav-select", handleNavSelect)
    return () => {
      window.clearTimeout(settleTimer)
      window.removeEventListener("scroll", handleScroll)
      window.removeEventListener("laci:nav-select", handleNavSelect)
    }
  }, [pathname])

  function handleNavClick(
    e: React.MouseEvent<HTMLAnchorElement>,
    id: SectionId,
    fromMobile = false
  ) {
    e.preventDefault()
    if (fromMobile) {
      setMobileOpen(false)
    }
    if (pathname === "/") {
      setActiveSection(id)
      document.documentElement.setAttribute("data-section", id)
      sessionStorage.setItem("laci_nav_section", id)
      const willBeScrolled = id !== "home"
      sessionStorage.setItem("laci_nav_scrolled", willBeScrolled ? "1" : "0")
      if (fromMobile) {
        window.setTimeout(() => {
          scrollToHomeSection(id)
        }, 120)
      } else {
        scrollToHomeSection(id)
      }
    } else {
      sessionStorage.setItem("laci_scroll_target", id)
      router.push("/")
    }
  }

  return (
    <>
      {/* Reserve the fixed header's space to prevent content shift. */}
      <div aria-hidden="true" className="h-[68px] sm:h-[84px]" />

      <header
        className={cn(
          "laci-site-header fixed inset-x-0 top-0 z-50 w-full",
          isScrolled
            ? "border-b border-slate-200/80 bg-white/95 px-0 py-0 shadow-[0_2px_16px_rgba(15,23,42,0.05)] backdrop-blur-md"
            : "px-3 pt-2.5 pb-1.5 sm:px-6 sm:pt-3 sm:pb-2 lg:px-8"
        )}
      >
        <div
          className={cn(
            "laci-site-header-inner mx-auto flex items-center justify-between",
            isScrolled
              ? "h-14 max-w-7xl rounded-none border-transparent bg-transparent px-4 shadow-none sm:h-16 sm:px-6 lg:px-10"
              : "h-14 max-w-6xl rounded-full border border-white/90 bg-white/95 px-3.5 shadow-[0_8px_30px_rgb(6,36,23,0.06)] backdrop-blur-md sm:h-[68px] sm:px-7"
          )}
        >
          <Link
            href="/"
            onClick={(e) => {
              if (pathname === "/") {
                handleNavClick(e, "home")
              }
            }}
            aria-label="Beranda Laci Digital"
            className="rounded-lg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#0a8744]"
          >
            <BrandMark />
          </Link>

          <nav
            aria-label="Navigasi Utama"
            className="hidden items-center gap-1.5 md:flex"
          >
            {sectionNavItems.map((item) => {
              const isActive = isHomePath && activeSection === item.id
              return (
                <Link
                  key={item.id}
                  href="/"
                  data-nav-id={isHomePath ? item.id : undefined}
                  onClick={(e) => handleNavClick(e, item.id)}
                  className={cn(
                    "laci-nav-pill rounded-full px-4 py-1.5 text-[13px] font-semibold transition-colors",
                    isActive
                      ? "border border-[#bce8ce] bg-[#e6f7ed] font-bold text-[#087038]"
                      : "border border-transparent text-slate-600 hover:bg-slate-100/80 hover:text-slate-900"
                  )}
                >
                  {item.label}
                </Link>
              )
            })}
          </nav>

          <div className="hidden items-center gap-4 md:flex">
            {isAuthenticated ? (
              <Button
                asChild
                className="h-10 rounded-full bg-[#0a8744] px-5 text-[13px] font-bold text-white shadow-md shadow-[#0a8744]/20 hover:bg-[#087038]"
              >
                <Link href="/dashboard">
                  Dashboard
                </Link>
              </Button>
            ) : (
              <>
                <Link
                  href="/login"
                  className="px-2 py-1.5 text-[13px] font-bold text-slate-700 transition-colors hover:text-[#0a8744] focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-[#0a8744]"
                >
                  Login
                </Link>
                <Button
                  asChild
                  className="h-10 gap-0 rounded-full bg-[#0a8744] px-5 text-[13px] font-bold text-white shadow-md shadow-[#0a8744]/20 hover:bg-[#087038]"
                >
                  <Link href="/register">
                    Daftar
                    <span
                      aria-hidden="true"
                      className={cn(
                        "laci-register-arrow inline-flex h-3.5 shrink-0 overflow-hidden",
                        isScrolled
                          ? "ml-0 w-0 -translate-x-1 opacity-0"
                          : "ml-2 w-3.5 translate-x-0 opacity-100"
                      )}
                    >
                      <ArrowRight className="size-3.5 shrink-0" />
                    </span>
                  </Link>
                </Button>
              </>
            )}
          </div>

          <div className="flex items-center md:hidden">
            <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
              <SheetTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  aria-label="Buka menu navigasi"
                  className="size-11 rounded-full text-slate-700 transition-transform hover:bg-slate-100/80 focus:outline-hidden focus-visible:border-[#0a8744] active:scale-95"
                >
                  <Menu className="size-5" />
                </Button>
              </SheetTrigger>
              <SheetContent
                side="right"
                className="p-0"
                onOpenAutoFocus={(e) => e.preventDefault()}
              >
                <SheetHeader className="border-b border-slate-100 px-5 pt-5 pb-4 pr-14 text-left">
                  <SheetTitle asChild>
                    <div>
                      <BrandMark />
                    </div>
                  </SheetTitle>
                  <SheetDescription className="mt-1 text-xs leading-relaxed text-slate-500">
                    Sistem Informasi &amp; Layanan Cerdas Administrasi PC IPNU
                    IPPNU Kabupaten Magetan.
                  </SheetDescription>
                </SheetHeader>

                <nav
                  aria-label="Navigasi Mobile"
                  className="flex flex-1 flex-col gap-1.5 overflow-y-auto overscroll-contain p-4"
                >
                  {sectionNavItems.map((item) => {
                    const isActive = isHomePath && activeSection === item.id
                    return (
                      <Link
                        key={item.id}
                        href="/"
                        onClick={(e) => handleNavClick(e, item.id, true)}
                        className={cn(
                          "flex items-center justify-between rounded-2xl px-4 py-3 text-sm font-semibold transition-colors",
                          isActive
                            ? "border border-[#bce8ce] bg-[#e6f7ed] font-bold text-[#087038]"
                            : "border border-transparent text-[#091526] hover:bg-slate-100/80"
                        )}
                      >
                        {item.label}
                      </Link>
                    )
                  })}
                </nav>

                <div className="mt-auto flex flex-col gap-2.5 border-t border-slate-100 bg-white p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
                  {isAuthenticated ? (
                    <SheetClose asChild>
                      <Button
                        asChild
                        className="h-11 w-full justify-center rounded-full bg-[#0a8744] text-sm font-bold text-white shadow-sm shadow-[#0a8744]/20 hover:bg-[#087038]"
                      >
                        <Link href="/dashboard">
                          Dashboard
                        </Link>
                      </Button>
                    </SheetClose>
                  ) : (
                    <>
                      <SheetClose asChild>
                        <Button
                          variant="outline"
                          asChild
                          className="h-11 w-full justify-center rounded-full border-[#64748b] bg-white text-sm font-bold text-[#091526] hover:bg-slate-50"
                        >
                          <Link href="/login">Login</Link>
                        </Button>
                      </SheetClose>
                      <SheetClose asChild>
                        <Button
                          asChild
                          className="h-11 w-full justify-center rounded-full bg-[#0a8744] text-sm font-bold text-white shadow-sm shadow-[#0a8744]/20 hover:bg-[#087038]"
                        >
                          <Link href="/register">
                            Daftar
                            <ArrowRight className="size-4" />
                          </Link>
                        </Button>
                      </SheetClose>
                    </>
                  )}
                </div>
              </SheetContent>
            </Sheet>
          </div>
        </div>
      </header>
    </>
  )
}

