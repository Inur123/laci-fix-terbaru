CREATE UNIQUE INDEX IF NOT EXISTS "Periode_one_active_per_user_idx"
ON "Periode" ("userId") WHERE "isActive" = true;

CREATE INDEX IF NOT EXISTS "Periode_user_active_idx"
ON "Periode" ("userId", "isActive") INCLUDE ("nama");

CREATE INDEX IF NOT EXISTS "ArsipSurat_user_periode_tanggal_idx"
ON "ArsipSurat" ("userId", "periodeId", "tanggal" DESC);

CREATE INDEX IF NOT EXISTS "BerkasPimpinan_user_periode_tanggal_idx"
ON "BerkasPimpinan" ("userId", "periodeId", "tanggal" DESC);

CREATE INDEX IF NOT EXISTS "BerkasSP_user_periode_tanggal_idx"
ON "BerkasSP" ("userId", "periodeId", "tanggalMulai" DESC);

CREATE INDEX IF NOT EXISTS "Anggota_user_periode_created_idx"
ON "Anggota" ("userId", "periodeId", "createdAt" DESC);

ALTER TABLE "Anggota"
ADD COLUMN IF NOT EXISTS "sourceAnggotaId" TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS "Anggota_unique_period_copy_idx"
ON "Anggota" ("userId", "periodeId", "sourceAnggotaId")
WHERE "sourceAnggotaId" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "AgendaKegiatan_user_periode_start_idx"
ON "AgendaKegiatan" ("userId", "periodeId", "tanggalMulai" DESC);

CREATE INDEX IF NOT EXISTS "PengajuanBerkas_cabang_status_created_idx"
ON "PengajuanBerkas" ("periodeId", "status", "createdAt" DESC);

CREATE INDEX IF NOT EXISTS "PengajuanBerkas_pac_status_created_idx"
ON "PengajuanBerkas" ("periodeIdPac", "userId", "status", "createdAt" DESC);

CREATE INDEX IF NOT EXISTS "Presensi_user_periode_tanggal_idx"
ON "Presensi" ("userId", "periodeId", "tanggal" DESC);

CREATE INDEX IF NOT EXISTS "LogEmail_status_created_idx"
ON "LogEmail" ("status", "createdAt" DESC);

CREATE INDEX IF NOT EXISTS "Session_expiresAt_idx"
ON "Session" ("expiresAt");
