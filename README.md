# Laci Digital

Proyek ini memisahkan aplikasi lama menjadi `fe/` (Next.js untuk antarmuka dan
rendering) dan `be/` (Go untuk API, autentikasi, aturan bisnis, database, dan
penyimpanan). Tampilan dan alur pengguna tetap mengikuti `laci-next-js`.

## Data lama

Database lokal `laci_fix_terbaru` dipulihkan dari backup
`laci_db_2026-10-18_000001.sql.gz` yang diberikan pengguna. Restore dilakukan
ke database baru; database asal dan file backup tidak diubah. Migrasi indeks
`be/migrations/002_performance_and_integrity.sql` telah diterapkan dan diuji.
File `be/.env` lokal menunjuk database ini serta menggunakan `ENCRYPTION_KEY`
dari aplikasi lama agar data terenkripsi dapat dibaca. `fe/.env.local` dan
`be/.env` menyimpan konfigurasi layanan lama, tetapi tidak menggunakan URL
database lama. Kedua file tersebut diabaikan Git dan tidak boleh dipublikasikan.
Login BE menerima hash password scrypt dari Better Auth lama serta bcrypt untuk
akun baru, sehingga akun dari backup tidak perlu direset hanya karena migrasi.

Backup SQL menyimpan metadata lampiran, bukan isi file yang berada di R2.
Kredensial R2, SMTP, Google, dan reCAPTCHA lama sudah disalin ke env baru.
Menjalankan fitur unggah/hapus file akan menyentuh bucket R2 lama, dan fitur
email akan mengirim pesan sungguhan. Google OAuth masih memerlukan callback
`http://localhost:3000/api/auth/google/callback` dalam daftar URI redirect
yang diizinkan di Google Cloud untuk berjalan lokal. Sesi login lama tidak
dipakai oleh backend baru, sehingga pengguna perlu login ulang.

Template email utama kini berada di BE. Validitas OTP adalah 5 menit sesuai
aturan aplikasi lama; isi email menampilkan durasi yang sama. Kesetaraan penuh
dengan alur email lama belum dapat diklaim: lampiran berkas pada email pengajuan
dan tautan verifikasi otomatis saat mengganti alamat email belum dipindahkan.
Kedua alur itu perlu diselesaikan sebelum mengganti aplikasi lama di produksi.

## Menjalankan lokal

Pada terminal pertama: `cd be && go run ./cmd/api`.
Pada terminal kedua: `cd fe && npm ci && npm run dev`.
Frontend tersedia di `http://localhost:3000`, API di `http://127.0.0.1:8080`.
Keterangan konfigurasi dan migrasi database ada di README masing-masing folder.
