import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { PengajuanBerkasDetail } from "@/components/features/pengajuan-berkas/pengajuan-berkas-detail";
import type { PengajuanBerkas } from "@/types/domain";

export function ReferensiPengajuanDetail({
  pengajuan,
  canAccessFile,
}: {
  pengajuan: PengajuanBerkas;
  canAccessFile: boolean;
}) {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" asChild>
          <Link href="/dashboard/referensi-pengajuan">
            <ArrowLeft className="w-4 h-4" />
          </Link>
        </Button>
        <div>
          <h2 className="text-xl font-semibold">Detail Pengajuan Surat</h2>
          <p className="text-sm text-muted-foreground">
            Detail informasi pengajuan surat
          </p>
        </div>
      </div>

      <PengajuanBerkasDetail
        pengajuan={pengajuan}
        isCabang={false}
        showSubmitterInfo={true}
        showActions={false}
        showFileAccess={canAccessFile}
      />
    </div>
  );
}
