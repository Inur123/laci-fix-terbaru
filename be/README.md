# Backend Laci Digital

Backend Laci Digital menyediakan API dan aturan bisnis untuk seluruh modul
administrasi PAC dan Cabang. Backend menangani autentikasi, otorisasi,
pengolahan data, realtime, penyimpanan file, email, dan backup database.

## Tanggung Jawab

- Memvalidasi request dan membentuk response JSON yang konsisten.
- Membatasi akses berdasarkan sesi, peran, pemilik data, dan periode.
- Menjalankan transaksi untuk operasi yang mengubah beberapa tabel.
- Mengenkripsi data sensitif, file, dan backup database.
- Mengelola upload, download, dan penghapusan lampiran pada object storage.
- Mengirim OTP, pemberitahuan pengajuan, dan email operasional.
- Menerbitkan pembaruan realtime tanpa menghambat request utama.
- Mencatat aktivitas pengguna dan hasil pengiriman email.

## Struktur

| Lokasi | Isi |
| --- | --- |
| `cmd/api` | Entry point layanan HTTP. |
| `internal/config` | Pembacaan dan validasi konfigurasi. |
| `internal/domain` | Model domain yang digunakan lintas modul. |
| `internal/httpapi` | Routing, middleware, handler, aturan bisnis, dan kontrak API. |
| `internal/security` | Enkripsi, password, token, dan proteksi backup. |
| `internal/storage` | Integrasi object storage. |
| `migrations` | Definisi schema, indeks, dan aturan integritas database. |

## Konvensi API

Endpoint menggunakan method dan status HTTP sesuai jenis operasinya. Response
error berisi status keberhasilan, pesan yang aman, dan kode error yang dapat
dipakai frontend. Endpoint daftar mendukung pagination serta filter dan sorting
sesuai kebutuhan modul.

Urutan bawaan menggunakan `createdAt` terbaru dengan ID sebagai pemecah seri.
Aturan ini menjaga hasil pagination tetap stabil ketika beberapa data memiliki
waktu pembuatan yang sama.

## Aturan Akses

Sekretaris PAC hanya dapat mengakses data miliknya dan data referensi yang
memang tersedia bagi PAC. Sekretaris Cabang dapat mengelola data Cabang,
memeriksa pengajuan PAC, dan membaca data wilayah pada endpoint yang memang
ditujukan untuk monitoring.

Penyebutan email atau ID pengguna dalam request tidak memberikan hak akses.
Backend selalu menentukan identitas dari sesi yang sudah diverifikasi.

## Pengelolaan Resource

Request database membawa context dan timeout agar pembatalan request ikut
menghentikan pekerjaan terkait. Operasi lintas tabel menggunakan transaksi.
Connection pool membatasi pemakaian koneksi dan menghindari pembuatan koneksi
baru pada setiap request.

Realtime menggunakan subscriber non-blocking. Subscriber lambat tidak menahan
request API atau subscriber lain. Backup dan retry email memakai penguncian agar
pekerjaan yang sama tidak berjalan bersamaan.

## Pengujian

Backend memiliki unit test, integration test dengan schema terisolasi, pengujian
otorisasi, transaksi, enkripsi, realtime, concurrency, race detection, dan
benchmark untuk jalur yang sensitif terhadap performa.
