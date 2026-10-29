export type Role = "SEKRETARIS_CABANG" | "SEKRETARIS_PAC";
export type LogAction =
  | "CREATE"
  | "UPDATE"
  | "DELETE"
  | "EXPORT"
  | "APPROVE"
  | "REJECT"
  | "LOGIN"
  | "LOGOUT";
export type LogModule =
  | "ARSIP_SURAT"
  | "ANGGOTA"
  | "BERKAS_PIMPINAN"
  | "BERKAS_SP"
  | "AGENDA_KEGIATAN"
  | "PENGAJUAN_BERKAS"
  | "PERIODE"
  | "USER"
  | "AUTH"
  | "PRESENSI";
export type PenerimaSurat = "IPNU" | "IPPNU" | "BERSAMA" | "CBP_KPP";
export type StatusPengajuan = "PENDING" | "DITERIMA" | "DITOLAK";

export type PengajuanBerkas = {
  id: string;
  userId: string;
  periodeId: string;
  periodeIdPac: string;
  noSurat: string;
  penerima: PenerimaSurat;
  tanggal: Date | string;
  keperluan: string;
  deskripsi: string | null;
  file: string | null;
  status: StatusPengajuan;
  alasanPenolakan: string | null;
  createdAt: Date | string;
  updatedAt: Date | string;
  user: {
    name: string;
    email: string;
  };
  periodePac: {
    id: string;
    nama: string;
  };
  periodeCabang: {
    id: string;
    nama: string;
  };
};

export type PengajuanBerkasStats = {
  total: number;
  ipnu: number;
  ippnu: number;
  bersama: number;
  cbpKpp: number;
  pending: number;
  diterima: number;
  ditolak: number;
};

export type Periode = {
  id: string;
  nama: string;
  userId: string;
  isActive: boolean;
  createdAt: Date | string;
  updatedAt: Date | string;
};

export type ArsipSurat = {
  id: string;
  userId: string;
  periodeId: string;
  noSurat: string;
  jenisSurat: "MASUK" | "KELUAR";
  tanggal: Date | string;
  pengirimPenerima: string;
  deskripsi: string | null;
  file: string | null;
  perihal: string;
  organisasi: "IPNU" | "IPPNU" | "BERSAMA" | "CBP_KPP" | null;
  createdAt: Date | string;
  updatedAt: Date | string;
};
