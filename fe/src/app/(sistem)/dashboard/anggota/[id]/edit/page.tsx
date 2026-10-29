import { getAnggotaById } from "@/app/actions/anggota-actions";
import { AnggotaForm } from "@/components/features/anggota/anggota-form";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Metadata } from "next";
import { Suspense } from "react";
import { AnggotaFormSkeleton } from "@/components/features/anggota/anggota-skeleton";

export const metadata: Metadata = {
  title: "Edit Anggota",
};

export default async function EditAnggotaPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  return (
    <Suspense fallback={<AnggotaFormSkeleton />}>
      <EditAnggotaContent params={params} />
    </Suspense>
  );
}

import { auth } from "@/auth";

async function EditAnggotaContent({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const session = await auth();
  const { id } = await params;
  const anggota = await getAnggotaById(id);

  if (!anggota) {
    return notFound();
  }

  return (
    <div className="space-y-8">
      <div className="flex items-center gap-4">
        <Button
          variant="outline"
          size="icon"
          asChild
          className="min-h-11 min-w-11"
        >
          <Link
            href={`/dashboard/anggota/${id}`}
            aria-label="Kembali ke detail anggota"
          >
            <ArrowLeft className="w-4 h-4" />
          </Link>
        </Button>

        <div className="min-w-0">
          <h2 className="text-xl font-semibold">Edit Data Anggota</h2>
          <p className="break-words text-sm text-muted-foreground [overflow-wrap:anywhere]">
            Lakukan perubahan pada profil anggota: {anggota.namaLengkap}
          </p>
        </div>
      </div>

      <AnggotaForm anggota={anggota} userRole={session?.user?.role} />
    </div>
  );
}
