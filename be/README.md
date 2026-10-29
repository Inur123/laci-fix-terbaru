# Backend Laci Digital (Go)

Backend ini menjadi satu-satunya pemilik akses database dan aturan bisnis.
Frontend Next.js berkomunikasi melalui HTTP API dan tidak lagi memakai Prisma.

## Menjalankan lokal

1. Untuk proyek lokal ini, `.env` sudah tersedia. Saat menyiapkan lingkungan
   lain, gunakan `DATABASE_URL` milik database hasil restore, **bukan** URL
   database lama. `ENCRYPTION_KEY` harus sama dengan aplikasi lama agar data
   terenkripsi tetap terbaca.
2. Jalankan `go mod download`.
3. Untuk database kosong, jalankan `migrations/001_baseline.sql` lalu
   `migrations/002_performance_and_integrity.sql`. Untuk database hasil restore
   dari backup aplikasi lama, jalankan **hanya** migrasi `002` setelah membuat
   salinan cadangan dan memeriksa tidak ada lebih dari satu periode aktif per
   pengguna. Jangan jalankan baseline di database yang sudah berisi data.
4. Jalankan `go run ./cmd/api`.

API tersedia di `http://127.0.0.1:8080`, health check di `/healthz`.

## Kontrak baca data untuk integrasi

Endpoint arsip `GET /api/arsip` memakai sesi pengguna yang aktif. Hasilnya
selalu dibatasi ke arsip milik pengguna dan periode yang dipilih; menyebut email
atau ID pengguna lain tidak memberi hak untuk membaca datanya. Contoh untuk
surat keluar organisasi bersama yang terbaru:

`GET /api/arsip?jenis=KELUAR&organisasi=BERSAMA&sortKey=tanggal&sortDir=desc&page=1&limit=1`

Respons berisi `data`, `total`, dan `totalPages`. Filter `jenis` dan
`organisasi` diterapkan sebelum pagination. Urutan tanggal memiliki pemecah
seri berdasarkan ID sehingga halaman stabil. Filter/sort metadata umum
dijalankan di PostgreSQL; pencarian `q` dan urutan kolom terenkripsi harus
mendekripsi data lebih dulu, sehingga lebih mahal pada volume besar. Spesifikasi
OpenAPI yang sudah diverifikasi tersedia di `GET /api/docs`; cakupannya saat ini
baru endpoint baca arsip, bukan seluruh API.

Belum ada token/role khusus AI. Jika nanti agen membaca atas nama pengguna,
identitas dan izin harus diverifikasi server-side; email yang disebut di prompt
tidak boleh diperlakukan sebagai bukti hak akses.

## Prinsip kompatibilitas

- Nama tabel dan kolom mengikuti schema Prisma lama, termasuk huruf besar.
- Enkripsi teks/file kompatibel dengan AES-256-CBC + scrypt milik aplikasi lama.
- Login menerima hash password scrypt Better Auth 1.5.6 dari database lama dan
  hash bcrypt untuk akun yang dibuat atau diubah melalui backend Go.
- Session disimpan di tabel `Session` yang sudah ada.
- Cookie browser bersifat HTTP-only dan dapat diteruskan oleh frontend saat SSR.
- Sesi lama tetap tersimpan sebagai data historis, tetapi tokennya tidak diterima
  oleh backend baru. Semua pengguna login ulang setelah migrasi.

## Backup berisi data lama

File `.sql.gz` adalah dump PostgreSQL biasa. Pulihkan ke **database baru yang
kosong**, bukan di atas database lama. Dump yang diuji memuat pemilik role
`laci_user`; bila role itu tidak ada di mesin lokal, abaikan perintah `OWNER TO`
dan `GRANT` saat restore, bukan mengubah data tabel. Jangan masukkan backup
atau `.env` ke Git.

Untuk menyiapkan `.env` lokal dari kunci enkripsi aplikasi lama tanpa menyalin
kredensial R2, email, atau Google, tersedia perintah:

`go run ./cmd/setup-local-env -source /path/to/laci-next-js/.env -database 'postgresql://user@localhost:5432/nama_database?sslmode=disable'`

Perintah ini tidak menimpa `.env` yang sudah ada. Database lokal yang telah
dibuat untuk proyek ini bernama `laci_fix_terbaru`.

Untuk menyalin kredensial layanan dari env aplikasi lama ke `.env` backend dan
`../fe/.env.local` tanpa mengubah URL database maupun session secret baru,
jalankan `go run ./cmd/sync-legacy-env -source /path/to/laci-next-js/.env`.
Perintah ini sudah dijalankan untuk proyek lokal ini. Layanan eksternal tidak
diuji dengan mengirim email atau mengubah objek R2.
