# Laci Digital

Laci Digital adalah sistem administrasi organisasi IPNU dan IPPNU untuk tingkat
PAC dan Cabang. Sistem ini mengelola arsip, anggota, pengajuan berkas, periode
kepengurusan, agenda, presensi, aktivitas pengguna, email, dan backup database.

## Peran Pengguna

| Peran | Cakupan |
| --- | --- |
| Sekretaris PAC | Mengelola data PAC miliknya, membuat pengajuan berkas, dan membaca referensi pengajuan yang telah diterima. |
| Sekretaris Cabang | Mengelola administrasi Cabang, memverifikasi pengajuan PAC, memantau data wilayah, serta mengelola pengguna dan operasional sistem. |

Hak akses diterapkan pada antarmuka dan API. Pengguna hanya dapat membuka data
yang sesuai dengan peran, kepemilikan, dan periode yang dipilih.

## Modul Sistem

| Modul | Fungsi |
| --- | --- |
| Dashboard | Menampilkan ringkasan administrasi pribadi dan monitoring wilayah. |
| Arsip Surat | Menyimpan surat masuk dan keluar berdasarkan organisasi, jenis surat, serta periode. |
| Berkas Pimpinan | Menyimpan dokumen pimpinan beserta catatan dan lampirannya. |
| Berkas SP | Mengelola dokumen SP IPNU atau IPPNU pada tingkat Cabang. |
| Pengajuan Berkas | Menangani pengajuan PAC, pemeriksaan Cabang, penerimaan, penolakan, dan pemberitahuan email. |
| Referensi Pengajuan | Menampilkan pengajuan yang telah diterima sebagai referensi bagi PAC. |
| Data Anggota | Mengelola identitas anggota, riwayat kaderisasi, pendidikan, foto, dan perpindahan data antarperiode. |
| Periode | Mengatur periode kepengurusan aktif serta akses ke data periode sebelumnya. |
| Agenda Kegiatan | Mengelola jadwal kegiatan Cabang dalam tampilan daftar dan kalender. |
| Presensi | Mengatur sesi presensi, jadwal buka dan tutup, peserta, serta ekspor daftar kehadiran. |
| Manajemen User | Mengelola akun PAC, status akun, detail data, dan reset password. |
| Riwayat Aktivitas | Mencatat tindakan pengguna dan menyediakan monitoring aktivitas. |
| Log Email | Mencatat status pengiriman email dan mendukung pengiriman ulang ketika diperlukan. |
| Backup Database | Membuat, menyimpan, mengunduh, dan merotasi backup database terenkripsi. |

## Pengelolaan Data

- Daftar menampilkan data terbaru berdasarkan waktu penambahan.
- Pencarian, filter, sorting, dan pagination diterapkan pada modul yang relevan.
- Data anggota tetap terikat pada pemilik dan periode kepengurusan.
- Pengajuan hanya dapat diproses melalui status dan peran yang sesuai.
- Presensi mengikuti tanggal, jam, dan status sesi yang ditentukan.
- Lampiran tersimpan di object storage dan diakses melalui token sementara.
- Modul yang mendukung pelaporan menyediakan ekspor Excel tanpa fitur impor.

## Arsitektur

Antarmuka dibangun dengan Next.js. Backend Go menyediakan API, autentikasi,
aturan bisnis, realtime, pengolahan data, email, penyimpanan file, dan backup.
PostgreSQL menyimpan data utama, sedangkan lampiran disimpan pada object storage.

## Keamanan

Sistem menggunakan sesi berbasis cookie HTTP-only, pembatasan akses berdasarkan
peran, validasi input, pembatasan kepemilikan data, enkripsi data sensitif,
enkripsi file, dan backup terenkripsi. Aktivitas penting dicatat untuk kebutuhan
audit dan pemantauan operasional.
