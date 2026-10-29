-- CreateSchema
CREATE SCHEMA IF NOT EXISTS "public";

-- CreateEnum
CREATE TYPE "Role" AS ENUM ('SEKRETARIS_CABANG', 'SEKRETARIS_PAC');

-- CreateEnum
CREATE TYPE "JenisSurat" AS ENUM ('MASUK', 'KELUAR');

-- CreateEnum
CREATE TYPE "Organisasi" AS ENUM ('IPNU', 'IPPNU', 'BERSAMA', 'CBP_KPP');

-- CreateEnum
CREATE TYPE "StatusPengajuan" AS ENUM ('PENDING', 'DITERIMA', 'DITOLAK');

-- CreateEnum
CREATE TYPE "PenerimaSurat" AS ENUM ('IPNU', 'IPPNU', 'BERSAMA', 'CBP_KPP');

-- CreateEnum
CREATE TYPE "JenisKelamin" AS ENUM ('LAKI_LAKI', 'PEREMPUAN');

-- CreateEnum
CREATE TYPE "LogAction" AS ENUM ('CREATE', 'UPDATE', 'DELETE', 'EXPORT', 'APPROVE', 'REJECT', 'LOGIN', 'LOGOUT');

-- CreateEnum
CREATE TYPE "LogModule" AS ENUM ('ARSIP_SURAT', 'ANGGOTA', 'BERKAS_PIMPINAN', 'BERKAS_SP', 'AGENDA_KEGIATAN', 'PENGAJUAN_BERKAS', 'PERIODE', 'USER', 'AUTH', 'PRESENSI');

-- CreateEnum
CREATE TYPE "EmailType" AS ENUM ('VERIFICATION', 'VERIFIED_SUCCESS', 'PENGAJUAN_USER', 'PENGAJUAN_ADMIN', 'PENGAJUAN_STATUS');

-- CreateEnum
CREATE TYPE "EmailStatus" AS ENUM ('PENDING', 'SENT', 'FAILED');

-- CreateTable
CREATE TABLE "User" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "emailVerified" BOOLEAN NOT NULL DEFAULT false,
    "image" TEXT,
    "role" "Role" NOT NULL DEFAULT 'SEKRETARIS_PAC',
    "isActive" BOOLEAN NOT NULL DEFAULT false,
    "periodeAktifId" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "lastLogoutAt" TIMESTAMP(3),

    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Periode" (
    "id" TEXT NOT NULL,
    "nama" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "isActive" BOOLEAN NOT NULL DEFAULT false,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Periode_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "ArsipSurat" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "noSurat" TEXT NOT NULL,
    "jenisSurat" "JenisSurat" NOT NULL,
    "tanggal" TIMESTAMP(3) NOT NULL,
    "pengirimPenerima" TEXT NOT NULL,
    "deskripsi" TEXT,
    "file" TEXT,
    "perihal" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "organisasi" "Organisasi",

    CONSTRAINT "ArsipSurat_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "BerkasSP" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "nama" TEXT NOT NULL,
    "tanggalMulai" TIMESTAMP(3) NOT NULL,
    "tanggalBerakhir" TIMESTAMP(3) NOT NULL,
    "catatan" TEXT,
    "file" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "organisasi" "Organisasi" NOT NULL,

    CONSTRAINT "BerkasSP_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "BerkasPimpinan" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "nama" TEXT NOT NULL,
    "tanggal" TIMESTAMP(3) NOT NULL,
    "catatan" TEXT,
    "file" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "BerkasPimpinan_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "PengajuanBerkas" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "periodeIdPac" TEXT NOT NULL,
    "noSurat" TEXT NOT NULL,
    "penerima" "PenerimaSurat" NOT NULL,
    "tanggal" TIMESTAMP(3) NOT NULL,
    "keperluan" TEXT NOT NULL,
    "deskripsi" TEXT,
    "file" TEXT NOT NULL,
    "status" "StatusPengajuan" NOT NULL DEFAULT 'PENDING',
    "alasanPenolakan" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "PengajuanBerkas_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Presensi" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "namaKegiatan" TEXT NOT NULL,
    "tempat" TEXT NOT NULL,
    "penyelenggara" TEXT NOT NULL,
    "tanggal" TIMESTAMP(3) NOT NULL,
    "jamMulai" TEXT NOT NULL,
    "jamSelesai" TEXT NOT NULL,
    "isActive" BOOLEAN NOT NULL DEFAULT true,
    "isForcedOpen" BOOLEAN NOT NULL DEFAULT false,
    "forcedOpenAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Presensi_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "PresensiData" (
    "id" TEXT NOT NULL,
    "presensiId" TEXT NOT NULL,
    "namaLengkap" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "noHp" TEXT NOT NULL,
    "emailHash" TEXT NOT NULL,
    "noHpHash" TEXT NOT NULL,
    "organisasi" TEXT NOT NULL,
    "tingkat" TEXT,
    "jabatan" TEXT,
    "instansi" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "PresensiData_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Anggota" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "nik" TEXT,
    "nia" TEXT,
    "email" TEXT,
    "foto" TEXT,
    "namaLengkap" TEXT NOT NULL,
    "tempatLahir" TEXT,
    "tanggalLahir" TIMESTAMP(3),
    "jenisKelamin" "JenisKelamin" NOT NULL,
    "alamatLengkap" TEXT,
    "noHp" TEXT,
    "hobi" TEXT,
    "jabatan" TEXT,
    "noRfid" TEXT,
    "pekerjaan" TEXT,
    "jenjangPendidikan" TEXT,
    "namaInstansiPendidikan" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Anggota_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Pendidikan" (
    "id" TEXT NOT NULL,
    "anggotaId" TEXT NOT NULL,
    "jenjang" TEXT NOT NULL,
    "namaSekolah" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Pendidikan_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Perkaderan" (
    "id" TEXT NOT NULL,
    "anggotaId" TEXT NOT NULL,
    "namaPerkaderan" TEXT NOT NULL,
    "tanggal" TIMESTAMP(3) NOT NULL,
    "tempat" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Perkaderan_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AgendaKegiatan" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "judul" TEXT NOT NULL,
    "deskripsi" TEXT,
    "lokasi" TEXT,
    "warna" TEXT NOT NULL,
    "tanggalMulai" TIMESTAMP(3) NOT NULL,
    "tanggalSelesai" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "AgendaKegiatan_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "AllowedOrigin" (
    "id" TEXT NOT NULL,
    "domain" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "AllowedOrigin_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "LogActivity" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "periodeId" TEXT NOT NULL,
    "action" "LogAction" NOT NULL,
    "module" "LogModule" NOT NULL,
    "description" TEXT NOT NULL,
    "entityId" TEXT,
    "ipAddress" TEXT,
    "userAgent" TEXT,
    "browser" TEXT,
    "device" TEXT,
    "location" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "LogActivity_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "LogEmail" (
    "id" TEXT NOT NULL,
    "to" TEXT NOT NULL,
    "subject" TEXT NOT NULL,
    "type" "EmailType" NOT NULL,
    "status" "EmailStatus" NOT NULL DEFAULT 'PENDING',
    "errorMessage" TEXT,
    "retryCount" INTEGER NOT NULL DEFAULT 0,
    "metadata" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "LogEmail_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Session" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "expiresAt" TIMESTAMP(3) NOT NULL,
    "token" TEXT NOT NULL,
    "ipAddress" TEXT,
    "userAgent" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Session_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Account" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "accountId" TEXT NOT NULL,
    "providerId" TEXT NOT NULL,
    "accessToken" TEXT,
    "refreshToken" TEXT,
    "idToken" TEXT,
    "expiresAt" TIMESTAMP(3),
    "accessTokenExpiresAt" TIMESTAMP(3),
    "refreshTokenExpiresAt" TIMESTAMP(3),
    "scope" TEXT,
    "password" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Account_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Verification" (
    "id" TEXT NOT NULL,
    "identifier" TEXT NOT NULL,
    "value" TEXT NOT NULL,
    "expiresAt" TIMESTAMP(3) NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Verification_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");

-- CreateIndex
CREATE INDEX "User_role_isActive_emailVerified_idx" ON "User"("role", "isActive", "emailVerified");

-- CreateIndex
CREATE INDEX "User_createdAt_idx" ON "User"("createdAt");

-- CreateIndex
CREATE INDEX "Periode_userId_idx" ON "Periode"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "Periode_nama_userId_key" ON "Periode"("nama", "userId");

-- CreateIndex
CREATE INDEX "ArsipSurat_userId_periodeId_idx" ON "ArsipSurat"("userId", "periodeId");

-- CreateIndex
CREATE INDEX "ArsipSurat_tanggal_idx" ON "ArsipSurat"("tanggal");

-- CreateIndex
CREATE INDEX "ArsipSurat_createdAt_idx" ON "ArsipSurat"("createdAt");

-- CreateIndex
CREATE INDEX "BerkasSP_userId_periodeId_idx" ON "BerkasSP"("userId", "periodeId");

-- CreateIndex
CREATE INDEX "BerkasPimpinan_userId_periodeId_idx" ON "BerkasPimpinan"("userId", "periodeId");

-- CreateIndex
CREATE INDEX "PengajuanBerkas_userId_status_idx" ON "PengajuanBerkas"("userId", "status");

-- CreateIndex
CREATE INDEX "PengajuanBerkas_periodeId_idx" ON "PengajuanBerkas"("periodeId");

-- CreateIndex
CREATE INDEX "PengajuanBerkas_periodeIdPac_idx" ON "PengajuanBerkas"("periodeIdPac");

-- CreateIndex
CREATE INDEX "Presensi_userId_periodeId_idx" ON "Presensi"("userId", "periodeId");

-- CreateIndex
CREATE INDEX "PresensiData_presensiId_idx" ON "PresensiData"("presensiId");

-- CreateIndex
CREATE UNIQUE INDEX "PresensiData_presensiId_emailHash_key" ON "PresensiData"("presensiId", "emailHash");

-- CreateIndex
CREATE UNIQUE INDEX "PresensiData_presensiId_noHpHash_key" ON "PresensiData"("presensiId", "noHpHash");

-- CreateIndex
CREATE INDEX "Anggota_userId_periodeId_idx" ON "Anggota"("userId", "periodeId");

-- CreateIndex
CREATE INDEX "Pendidikan_anggotaId_idx" ON "Pendidikan"("anggotaId");

-- CreateIndex
CREATE INDEX "Perkaderan_anggotaId_idx" ON "Perkaderan"("anggotaId");

-- CreateIndex
CREATE INDEX "AgendaKegiatan_userId_periodeId_idx" ON "AgendaKegiatan"("userId", "periodeId");

-- CreateIndex
CREATE INDEX "AgendaKegiatan_tanggalMulai_idx" ON "AgendaKegiatan"("tanggalMulai");

-- CreateIndex
CREATE UNIQUE INDEX "AllowedOrigin_domain_key" ON "AllowedOrigin"("domain");

-- CreateIndex
CREATE INDEX "LogActivity_userId_periodeId_createdAt_idx" ON "LogActivity"("userId", "periodeId", "createdAt" DESC);

-- CreateIndex
CREATE INDEX "LogActivity_module_createdAt_idx" ON "LogActivity"("module", "createdAt" DESC);

-- CreateIndex
CREATE INDEX "LogActivity_userId_createdAt_idx" ON "LogActivity"("userId", "createdAt" DESC);

-- CreateIndex
CREATE INDEX "LogActivity_createdAt_idx" ON "LogActivity"("createdAt" DESC);

-- CreateIndex
CREATE INDEX "LogEmail_status_idx" ON "LogEmail"("status");

-- CreateIndex
CREATE INDEX "LogEmail_type_idx" ON "LogEmail"("type");

-- CreateIndex
CREATE INDEX "LogEmail_createdAt_idx" ON "LogEmail"("createdAt" DESC);

-- CreateIndex
CREATE INDEX "LogEmail_to_createdAt_idx" ON "LogEmail"("to", "createdAt" DESC);

-- CreateIndex
CREATE UNIQUE INDEX "Session_token_key" ON "Session"("token");

-- CreateIndex
CREATE INDEX "Session_userId_idx" ON "Session"("userId");

-- CreateIndex
CREATE INDEX "Session_token_idx" ON "Session"("token");

-- CreateIndex
CREATE INDEX "Account_userId_idx" ON "Account"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "Account_providerId_accountId_key" ON "Account"("providerId", "accountId");

-- CreateIndex
CREATE INDEX "Verification_identifier_idx" ON "Verification"("identifier");

-- CreateIndex
CREATE UNIQUE INDEX "Verification_identifier_value_key" ON "Verification"("identifier", "value");

-- AddForeignKey
ALTER TABLE "Periode" ADD CONSTRAINT "Periode_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ArsipSurat" ADD CONSTRAINT "ArsipSurat_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "ArsipSurat" ADD CONSTRAINT "ArsipSurat_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "BerkasSP" ADD CONSTRAINT "BerkasSP_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "BerkasSP" ADD CONSTRAINT "BerkasSP_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "BerkasPimpinan" ADD CONSTRAINT "BerkasPimpinan_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "BerkasPimpinan" ADD CONSTRAINT "BerkasPimpinan_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PengajuanBerkas" ADD CONSTRAINT "PengajuanBerkas_periodeIdPac_fkey" FOREIGN KEY ("periodeIdPac") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PengajuanBerkas" ADD CONSTRAINT "PengajuanBerkas_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PengajuanBerkas" ADD CONSTRAINT "PengajuanBerkas_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Presensi" ADD CONSTRAINT "Presensi_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Presensi" ADD CONSTRAINT "Presensi_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "PresensiData" ADD CONSTRAINT "PresensiData_presensiId_fkey" FOREIGN KEY ("presensiId") REFERENCES "Presensi"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Anggota" ADD CONSTRAINT "Anggota_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Anggota" ADD CONSTRAINT "Anggota_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Pendidikan" ADD CONSTRAINT "Pendidikan_anggotaId_fkey" FOREIGN KEY ("anggotaId") REFERENCES "Anggota"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Perkaderan" ADD CONSTRAINT "Perkaderan_anggotaId_fkey" FOREIGN KEY ("anggotaId") REFERENCES "Anggota"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AgendaKegiatan" ADD CONSTRAINT "AgendaKegiatan_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "AgendaKegiatan" ADD CONSTRAINT "AgendaKegiatan_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "LogActivity" ADD CONSTRAINT "LogActivity_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "LogActivity" ADD CONSTRAINT "LogActivity_periodeId_fkey" FOREIGN KEY ("periodeId") REFERENCES "Periode"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Session" ADD CONSTRAINT "Session_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Account" ADD CONSTRAINT "Account_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;
