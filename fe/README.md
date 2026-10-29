# Frontend Laci Digital

Frontend Laci Digital menyediakan antarmuka administrasi untuk Sekretaris PAC
dan Sekretaris Cabang. Next.js menangani rendering halaman, navigasi,
pengelolaan sesi pada sisi aplikasi, serta komunikasi dengan backend melalui
server action dan proxy API.

## Tanggung Jawab

- Menampilkan menu dan aksi berdasarkan peran pengguna.
- Menyediakan formulir dengan validasi dan pesan kesalahan yang jelas.
- Menampilkan data dalam tabel yang mendukung pencarian, filter, sorting, dan pagination.
- Menangani upload lampiran dan menampilkan status proses kepada pengguna.
- Menyediakan tampilan desktop dan mobile tanpa mengubah alur kerja tiap modul.
- Menampilkan pembaruan data realtime pada halaman yang relevan.
- Menyediakan ekspor Excel pada modul yang memerlukan pelaporan.

## Struktur

| Lokasi | Isi |
| --- | --- |
| `src/app` | Halaman, layout, server action, dan route publik. |
| `src/components/features` | Komponen berdasarkan modul administrasi. |
| `src/components/layout` | Navigasi dan kerangka halaman dashboard. |
| `src/components/providers` | Session monitoring, realtime, dan reCAPTCHA. |
| `src/components/shared` | Dialog, crop gambar, filter pengguna, dan komponen lintas fitur. |
| `src/components/ui` | Komponen dasar antarmuka. |
| `src/lib` | Client backend, utilitas tanggal, file, presensi, dan ekspor Excel. |
| `src/types` | Tipe data yang digunakan frontend. |

## Perilaku Berdasarkan Peran

Sekretaris PAC melihat menu dan data milik PAC yang sedang login. Pengajuan yang
telah diterima dapat dibaca melalui menu referensi. Sekretaris Cabang mendapat
menu verifikasi pengajuan, agenda kegiatan, Berkas SP, manajemen user, log email,
monitoring wilayah, dan backup database.

Frontend tetap menyembunyikan menu yang tidak sesuai, tetapi backend menjadi
sumber keputusan akhir untuk setiap izin akses.

## Tampilan Data

Tabel menggunakan urutan data terbaru sebagai tampilan awal. Nilai panjang
dipotong secara visual tanpa mengubah isi data, dan nilai lengkap tetap tersedia
pada detail atau tooltip yang sesuai. Tabel lebar dapat digeser pada layar kecil
tanpa membuat halaman melebar.

Tombol aksi dibuat ringkas pada desktop dan memiliki area sentuh yang lebih
besar pada perangkat mobile. Form menampilkan indikator wajib pada label,
sedangkan kesalahan proses ditampilkan melalui notifikasi.

## Integrasi API

Frontend tidak mengakses database atau object storage secara langsung. Server
action meneruskan request ke backend bersama cookie sesi yang diperlukan.
Response API dipetakan ke state halaman tanpa mengubah aturan bisnis di sisi
client.

## Pengujian

Frontend memiliki pemeriksaan lint, typecheck, production build, pengujian
utilitas, serta QA visual untuk navigasi, form, tabel, responsivitas, dan aksi
berdasarkan peran.
