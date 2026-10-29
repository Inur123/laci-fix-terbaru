# Frontend Laci Digital

Frontend ini mempertahankan halaman, komponen, dan alur aplikasi Next.js lama.
Database, autentikasi, enkripsi, upload, dan aturan bisnis sekarang dijalankan
oleh API Go di `../be`. Next.js tetap melakukan rendering server dan meneruskan
permintaan `/api/*` ke backend.

## Menjalankan lokal

1. Pastikan backend Go sudah berjalan di `http://127.0.0.1:8080`.
2. Jalankan `npm ci` lalu `npm run dev` dari folder ini.
3. Buka `http://localhost:3000`.

`.env.local` sudah disinkronkan dari aplikasi lama. Jika alamat backend berubah,
sesuaikan `BACKEND_URL`. `API_KEY` di sini hanya dibaca oleh server Next.js.
Pada mode development lokal reCAPTCHA dilewati agar pembatasan domain lama
tidak menghalangi login; pada produksi, site key di frontend dan secret key di
backend tetap digunakan.

Folder FE tidak menyimpan schema/migrasi Prisma maupun template pengiriman
email lagi. Migrasi database dan pengiriman email dimiliki BE Go. Server action
di FE tetap dipakai sebagai penghubung halaman ke API Go, bukan akses database.
`src/app/api` juga tidak diperlukan: seluruh `/api/*` diteruskan ke BE melalui
rewrite di `next.config.ts`.

## Verifikasi

Jalankan `npm run lint`, `npm run typecheck`, dan `npm run build` sebelum rilis.
Untuk produksi, set `BACKEND_URL` ke alamat backend yang hanya dapat diakses
oleh frontend; jangan mengekspos port backend langsung ke internet.
