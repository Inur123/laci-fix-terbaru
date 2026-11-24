import { auth } from "@/auth";
import HomeClient from "@/components/features/home/home-client";
import { getPublicStats } from "@/app/actions/dashboard-actions";
import { publicFont } from "@/lib/public-font";

export const dynamic = "force-dynamic";

export default async function Home() {
  const session = await auth();

  //  Use cached stats
  const { anggotaCount, suratCount } = await getPublicStats();

  return (
    <div className={publicFont.className}>
      <HomeClient
        session={session}
        stats={{
          anggotaTerdaftar: anggotaCount,
          suratDiproses: suratCount,
        }}
      />
    </div>
  );
}

/**
 *  OPTIONAL: Manual cache revalidation function
 *
 * Usage in server action:
 * import { revalidateTag } from 'next/cache';
 * revalidateTag('public-stats');
 */
