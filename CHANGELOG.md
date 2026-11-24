# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v2.2.0] - 2026-11-24

### Added
- **Endpoint Publik Agenda Kegiatan (`GET /api/kegiatan`)**:
  - Endpoint publik tanpa autentikasi/cookie untuk integrasi widget kalender blog eksternal (Blogger FullCalendar).
  - Header CORS terbuka (`Access-Control-Allow-Origin: *`, `Methods: GET, OPTIONS`) serta penanganan preflight `OPTIONS` yang merespons `204 No Content`.
  - Format output JSON standar RFC3339 untuk `tanggal_mulai` dan `tanggal_selesai` (atau `null` jika tidak ada tanggal selesai).
  - Dekripsi aman AES-GCM untuk kolom `judul` dan `lokasi` (termasuk penanganan aman jika `lokasi` bernilai `null` atau kosong).
  - Pengujian menyeluruh (`agenda_public_test.go`) mencakup unit test CORS, serialisasi JSON, dan integration test pembacaan database.

---

## [v2.1.0] - 2026-11-24

### Added
- **Agenda Kegiatan Detail & Kalender Libur Nasional**:
  - Halaman publik detail kegiatan (`/agenda-kegiatan/[id]`) dengan status kegiatan dinamis dan badge badges.
  - Integrasi API hari libur nasional pada tampilan kalender FullCalendar interaktif.
  - Dialog interaktif klik tanggal kalender yang menampilkan daftar agenda pada tanggal terpilih secara rapi.
- **Database Migration**:
  - `003_add_asal_pimpinan.sql`: Penambahan kolom `asal_pimpinan` untuk mendukung fleksibilitas data pimpinan cabang/PAC.
- **Lisensi & Metadata Repositori**:
  - Penambahan lisensi resmi MIT License (`LICENSE`) dan dokumentasi lisensi pada `README.md`.
  - Penyelarasan deskripsi, website, dan topik repository GitHub.

### Changed
- **Desain Tombol Form & Action Diseragamkan**:
  - Standardisasi tombol aksi simpan dan batal: `[ ✕ Batal ]` (outline) dan `[ ⬆ Simpan ]` (upload icon) pada:
    - Arsip Surat (`arsip-surat-form.tsx`)
    - Pengajuan Berkas (`pengajuan-berkas-form.tsx`)
    - Berkas Pimpinan (`berkas-pimpinan-form.tsx`)
    - Berkas SP (`berkas-sp-form.tsx`)
    - Data Anggota (`anggota-form.tsx`)
    - Presensi Kegiatan (`presensi-form.tsx`)
    - Agenda Kegiatan (`kegiatan-form.tsx`)
    - Periode Kepengurusan (`add-periode-form.tsx`, `edit-periode-form.tsx`)
- **Responsive Table & Touch Target Standard**:
  - Optimalisasi tabel data di mobile dengan touch-target `h-11 w-11 p-0 sm:h-8 sm:w-8`.
  - Lebar kolom Aksi responsif (`w-[180px] sm:w-[130px]` untuk 3 tombol, `w-[100px] sm:w-[80px]` untuk 1 tombol) dengan padding rapi di mobile.

### Fixed
- **Penyelarasan Icon Search**:
  - Memperbaiki posisi vertikal icon `Search` di seluruh input pencarian agar selalu berada tepat di tengah (`pointer-events-none absolute left-3 top-1/2 -translate-y-1/2`) di semua resolusi dan mobile viewport.
- **Referensi Pengajuan**:
  - Perbaikan layout tabel mobile agar tombol detail/eye tidak terpotong atau mepet ke tepi layar.
- **Presensi & Sesi**:
  - Perbaikan penanganan token sesi dan tampilan konfirmasi presensi.

---

## [v2.0.0] - 2026-10-23

### Added
- **Major Architecture Overhaul (Generasi 2)**:
  - Migrasi arsitektur dari sistem awal monolitik [laci-next-js](https://github.com/Inur123/laci-next-js) menjadi sistem decoupled: Frontend Next.js modern (`fe`) dan Backend Go berkinerja tinggi (`be`).
- Inisialisasi awal arsitektur Next.js Frontend dan Go Backend untuk sistem administrasi Laci Digital (IPNU-IPPNU).
- Manajemen Arsip Surat (Masuk, Keluar, Bersama).
- Manajemen Pengajuan Berkas & Surat Rekomendasi/Pengesahan.
- Manajemen Data Anggota & Riwayat Kepengurusan.
- Presensi QR Code & Absensi Real-time.
- Manajemen User & Role-Based Access Control (Sekretaris Cabang & Sekretaris PAC).
- Sistem Enkripsi AES-GCM file penyimpanan Cloudflare R2.
- Backup Database otomatis dan audit log aktivitas.
