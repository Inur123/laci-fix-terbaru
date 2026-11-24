import type { ComponentType } from "react";

import { SiteFooter } from "@/components/features/public-v2/site-footer";
import { SiteHeader } from "@/components/features/public-v2/site-header";

interface LegalSection {
  title: string;
  content: string;
}

interface LegalPageShellProps {
  badge: string;
  title: string;
  description: string;
  updatedAt: string;
  icon: ComponentType<{ className?: string }>;
  sections: LegalSection[];
  isAuthenticated?: boolean;
}

export function LegalPageShell({
  badge,
  title,
  description,
  updatedAt,
  icon: Icon,
  sections,
  isAuthenticated = false,
}: LegalPageShellProps) {
  return (
    <div className="min-h-screen overflow-x-clip bg-[#fbfdfc] text-[#091526]">
      <div className="relative overflow-hidden bg-gradient-to-br from-[#e3f6eb] via-[#f3fbf6] to-[#dff4e7]">
        <div
          aria-hidden="true"
          className="absolute -right-20 top-10 size-72 rounded-full bg-[#bfe7cf]/45 blur-3xl"
        />
        <div
          aria-hidden="true"
          className="absolute -bottom-20 left-8 size-64 rounded-full bg-white/70 blur-2xl"
        />
        <SiteHeader isAuthenticated={isAuthenticated} />
        <div className="relative mx-auto max-w-4xl px-4 pb-14 pt-10 sm:px-6 sm:pb-18 sm:pt-14 lg:px-8">
          <div className="flex size-12 items-center justify-center rounded-2xl bg-white/85 text-[#0a8744] shadow-sm ring-1 ring-emerald-100">
            <Icon className="size-6" />
          </div>
          <p className="mt-5 text-xs font-extrabold uppercase tracking-[0.2em] text-[#0a8744]">
            {badge}
          </p>
          <h1 className="mt-3 text-3xl font-extrabold tracking-tight sm:text-5xl">
            {title}
          </h1>
          <p className="mt-3 text-sm font-medium text-slate-500">
            Terakhir diperbarui: {updatedAt}
          </p>
          <p className="mt-5 max-w-3xl text-sm leading-7 text-slate-600 sm:text-base">
            {description}
          </p>
        </div>
      </div>

      <main className="mx-auto max-w-4xl px-4 py-12 sm:px-6 sm:py-16 lg:px-8">
        <div className="space-y-5">
          {sections.map((section) => (
            <section
              key={section.title}
              className="rounded-[22px] border border-slate-200/80 bg-white p-5 shadow-[0_12px_35px_-26px_rgba(15,23,42,0.28)] sm:p-7"
            >
              <h2 className="text-base font-extrabold text-[#091526] sm:text-lg">
                {section.title}
              </h2>
              <div className="mt-3 whitespace-pre-line text-sm leading-7 text-slate-600">
                {section.content}
              </div>
            </section>
          ))}
        </div>
      </main>
      <SiteFooter />
    </div>
  );
}
