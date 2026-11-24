import { auth } from "@/auth";
import { getKegiatanById } from "@/app/actions/agenda-kegiatan-actions";
import { redirect, notFound } from "next/navigation";
import { KegiatanDetail } from "@/components/features/agenda-kegiatan/kegiatan-detail";
import { KegiatanDetailSkeleton } from "@/components/features/agenda-kegiatan/kegiatan-skeleton";
import { Suspense } from "react";
import { Metadata } from "next";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const kegiatan = await getKegiatanById(id);
  return {
    title: kegiatan
      ? `Detail: ${kegiatan.judul} | Laci Digital`
      : "Kegiatan Tidak Ditemukan",
  };
}

export default async function DetailKegiatanPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  return (
    <Suspense fallback={<KegiatanDetailSkeleton />}>
      <DetailKegiatanContent params={params} />
    </Suspense>
  );
}

async function DetailKegiatanContent({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const session = await auth();
  if (!session?.user) redirect("/login");

  const kegiatan = await getKegiatanById(id);

  if (!kegiatan) {
    notFound();
  }

  return <KegiatanDetail kegiatan={kegiatan} userRole={session.user.role} />;
}
