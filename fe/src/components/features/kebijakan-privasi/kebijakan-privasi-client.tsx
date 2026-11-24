import { ShieldCheck } from "lucide-react";

import { LegalPageShell } from "@/components/features/public-v2/legal-page-shell";

const sections = [
  {
    title: "1. Tentang Kebijakan Ini",
    content: `Kebijakan Privasi ini menjelaskan data yang diproses oleh Laci Digital, tujuan pemrosesan, pihak yang dapat mengaksesnya, serta pilihan yang tersedia bagi pengguna.

Laci Digital dikelola untuk kebutuhan administrasi PC IPNU IPPNU Kabupaten Magetan. Kebijakan ini berlaku bagi Sekretaris Cabang, Sekretaris PAC, anggota yang datanya dicatat, dan peserta yang mengisi presensi publik.`,
  },
  {
    title: "2. Data Akun",
    content: `Saat akun dibuat atau diperbarui, sistem dapat memproses:

• nama lengkap;
• alamat email;
• password dalam bentuk hash, bukan teks asli;
• peran dan status akun;
• foto profil;
• identitas organisasi, termasuk nama PAC;
• status verifikasi email dan waktu login terakhir.

Jika login dengan Google digunakan, sistem menerima identitas dasar yang disetujui melalui Google, seperti email, nama, dan foto profil. Login Google hanya tersedia bagi akun Laci Digital yang sudah terdaftar dan aktif.`,
  },
  {
    title: "3. Data Administrasi Organisasi",
    content: `Sesuai fitur yang digunakan, sistem dapat menyimpan:

• periode kepengurusan;
• arsip surat, berkas pimpinan, Berkas SP, agenda, dan pengajuan;
• nomor surat, perihal, tanggal, status, catatan, serta file lampiran;
• data anggota, seperti NIK, NIA, nama, email, nomor telepon, alamat, tempat dan tanggal lahir, jenis kelamin, jabatan, pekerjaan, pendidikan, instansi, riwayat pendidikan, dan riwayat kaderisasi;
• informasi PAC, Cabang, kegiatan, dan data organisasi lain yang dimasukkan pengguna.

Pengguna wajib membatasi data yang dimasukkan pada informasi yang memang diperlukan untuk administrasi organisasi.`,
  },
  {
    title: "4. Data Presensi",
    content: `Peserta dapat mengisi presensi melalui tautan atau QR publik. Data yang dapat diproses meliputi nama, email, nomor telepon, organisasi, tingkatan, jabatan, instansi, waktu kehadiran, dan kegiatan yang diikuti.

Email dan nomor telepon juga dipakai dalam bentuk hash untuk membantu mencegah pengisian ganda. Pengurus yang membuat presensi bertanggung jawab memastikan peserta mengetahui tujuan pengumpulan data tersebut.`,
  },
  {
    title: "5. Data Keamanan dan Aktivitas",
    content: `Untuk keamanan, audit, dan penanganan masalah, sistem mencatat informasi seperti:

• alamat IP;
• user agent, jenis browser, dan perangkat;
• alamat lokasi ringkas saat login;
• waktu, tindakan, modul, dan entitas yang diakses atau diubah;
• sesi login serta waktu kedaluwarsanya;
• status dan riwayat pengiriman email.

Browser meminta koordinat saat login. Koordinat dipakai untuk mencari alamat melalui layanan geolokasi dan disimpan sementara pada browser selama paling lama 24 jam. Log aktivitas menyimpan alamat lokasi ringkas, bukan password atau kode OTP.`,
  },
  {
    title: "6. Cara Data Diperoleh",
    content: `Data diperoleh dari:

• informasi yang dimasukkan langsung oleh pengguna atau peserta;
• akun Google saat pengguna memilih login dengan Google;
• browser dan perangkat saat login atau menggunakan sistem;
• aktivitas yang dilakukan melalui dashboard dan API;
• dokumen, foto, dan file yang diunggah;
• pengurus lain yang berwenang mengelola data organisasi.

Jika Anda memasukkan data orang lain, pastikan Anda memiliki kewenangan organisasi dan telah memberikan informasi yang diperlukan kepada orang tersebut.`,
  },
  {
    title: "7. Tujuan Pemrosesan",
    content: `Data digunakan untuk:

• membuat, memverifikasi, mengaktifkan, dan mengamankan akun;
• menjalankan administrasi arsip, anggota, pengajuan, agenda, presensi, dan periode;
• menerapkan hak akses Cabang dan PAC;
• mengirim OTP serta notifikasi operasional;
• menghasilkan ekspor dan backup yang diminta pengguna berwenang;
• mencatat audit untuk mencegah penyalahgunaan dan menelusuri perubahan;
• memperbaiki kesalahan, menjaga stabilitas, dan meningkatkan keamanan sistem.`,
  },
  {
    title: "8. Penyedia Layanan",
    content: `Laci Digital menggunakan layanan berikut sesuai kebutuhan fitur:

• Google OAuth untuk login dan Google reCAPTCHA untuk mengurangi penyalahgunaan form;
• Cloudflare R2 untuk penyimpanan file dan backup;
• penyedia SMTP untuk pengiriman OTP serta notifikasi;
• BigDataCloud dan OpenStreetMap Nominatim untuk mengubah koordinat login menjadi alamat;
• layanan hari libur untuk membantu tampilan kalender;
• penyedia hosting aplikasi dan database.

Data hanya dikirim sejauh yang diperlukan oleh fitur tersebut. Kebijakan privasi masing-masing penyedia juga berlaku ketika layanannya digunakan.`,
  },
  {
    title: "9. Cookie dan Penyimpanan Browser",
    content: `Sistem memakai cookie dan penyimpanan browser yang diperlukan untuk fungsi utama, antara lain:

• cookie sesi untuk mempertahankan login;
• cookie state sementara untuk melindungi alur login Google;
• lokasi dan alamat login yang berlaku paling lama 24 jam;
• pilihan periode yang sedang dilihat;
• preferensi antarmuka yang diperlukan oleh aplikasi.

Cookie sesi menggunakan pengaturan HttpOnly dan SameSite. Pada lingkungan produksi, cookie sesi juga dikirim melalui koneksi HTTPS. Memblokir cookie penting dapat membuat login, periode, atau fitur lain tidak berfungsi.`,
  },
  {
    title: "10. Keamanan Data",
    content: `Sistem menerapkan kontrol teknis dan operasional, termasuk:

• hash bcrypt untuk password;
• token sesi dan OTP yang disimpan dalam bentuk digest;
• enkripsi AES-GCM untuk data sensitif dan file baru yang ditetapkan terenkripsi;
• validasi input, pembatasan ukuran request, dan pemeriksaan file;
• autentikasi, otorisasi berbasis peran, dan pembatasan kepemilikan data;
• rate limit, validasi origin, dan pencatatan audit;
• backup baru yang dienkripsi.

Tidak ada sistem yang dapat menjamin keamanan mutlak. Pengguna tetap harus menjaga perangkat, email, password, dan sesi login miliknya.`,
  },
  {
    title: "11. Akses dan Pengungkapan",
    content: `Sekretaris PAC mengakses data dalam lingkup akun dan periodenya. Sekretaris Cabang dapat mengakses data lintas PAC melalui fitur Cabang yang tersedia untuk verifikasi, pengelolaan akun, administrasi, dan pengawasan organisasi.

Data dapat diungkapkan kepada pengurus yang berwenang, penyedia layanan yang diperlukan untuk menjalankan sistem, atau pihak yang diwajibkan oleh hukum. Data tidak digunakan untuk menjual profil anggota atau peserta kepada pihak lain.`,
  },
  {
    title: "12. Penyimpanan dan Penghapusan",
    content: `Data aktif disimpan selama akun, periode, atau kegiatan administrasi masih diperlukan. OTP berlaku 5 menit. Sesi berakhir saat kedaluwarsa, logout, akun dinonaktifkan, atau password direset sesuai mekanisme sistem.

Penghapusan akun atau periode menghapus data terkait dari database aktif melalui relasi yang tersedia. File terkait akan dihapus dari object storage saat proses penghapusan berhasil.

Backup database baru dienkripsi dan dibatasi hingga 10 backup terbaru. Data yang sudah dihapus dari sistem aktif dapat tetap ada dalam backup sampai backup tersebut keluar dari rotasi. Catatan keamanan dan log dapat disimpan selama diperlukan untuk audit serta penanganan insiden.`,
  },
  {
    title: "13. Pilihan dan Hak Anda",
    content: `Pengguna dapat memperbarui profil dan data yang berada dalam kewenangannya melalui dashboard. Untuk koreksi akun, penghapusan akun, akses yang salah, atau permintaan terkait data pribadi, hubungi Sekretaris Cabang atau pengelola Laci Digital.

Permintaan akan diperiksa berdasarkan identitas pemohon, kewenangan organisasi, kebutuhan administrasi, keamanan, serta kewajiban yang berlaku. Penghapusan data tertentu dapat ditolak atau ditunda jika data masih diperlukan untuk audit atau penyelesaian proses organisasi.`,
  },
  {
    title: "14. Perubahan dan Kontak",
    content: `Kebijakan ini dapat diperbarui saat fitur, penyedia layanan, atau praktik keamanan berubah. Tanggal pembaruan terakhir ditampilkan pada halaman ini.

Pertanyaan atau permintaan terkait privasi dapat disampaikan melalui:

Email: lacipelajarnumagetan@gmail.com
WhatsApp: +62 858-5051-2135
Lokasi: Magetan, Jawa Timur, Indonesia`,
  },
];

export default function KebijakanPrivasiClient({
  isAuthenticated = false,
}: {
  isAuthenticated?: boolean;
}) {
  return (
    <LegalPageShell
      badge="Privasi"
      title="Kebijakan Privasi"
      description="Penjelasan tentang data yang diproses Laci Digital, tujuan penggunaannya, penyedia layanan, keamanan, serta pilihan pengguna."
      updatedAt="30 Oktober 2026"
      icon={ShieldCheck}
      sections={sections}
      isAuthenticated={isAuthenticated}
    />
  );
}
