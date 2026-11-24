import { FileText } from "lucide-react";

import { LegalPageShell } from "@/components/features/public-v2/legal-page-shell";

const sections = [
  {
    title: "1. Ruang Lingkup",
    content: `Ketentuan ini berlaku saat Anda mengakses atau menggunakan Laci Digital, sistem administrasi internal PC IPNU IPPNU Kabupaten Magetan.

Dengan menggunakan Laci Digital, Anda menyatakan telah membaca dan menyetujui ketentuan ini serta Kebijakan Privasi. Jika tidak setuju, jangan membuat akun, login, atau memasukkan data ke sistem.`,
  },
  {
    title: "2. Pengguna dan Peran",
    content: `Laci Digital digunakan oleh pengurus yang diberi kewenangan sebagai Sekretaris Cabang atau Sekretaris PAC.

Sekretaris Cabang mengelola administrasi tingkat cabang, akun PAC, agenda kegiatan, Berkas SP, log email, dan backup database.

Sekretaris PAC mengelola data PAC-nya, termasuk anggota, arsip, berkas pimpinan, presensi, periode, serta pengajuan ke Cabang.

Peserta kegiatan dapat mengisi presensi melalui tautan publik atau QR tanpa memperoleh akses ke dashboard.`,
  },
  {
    title: "3. Pendaftaran, Verifikasi, dan Aktivasi",
    content: `Pendaftaran umum membuat akun Sekretaris PAC dalam keadaan belum aktif. Pengguna harus memverifikasi email dengan kode OTP 6 digit. Kode berlaku selama 5 menit dan kode baru menggantikan kode sebelumnya.

Verifikasi email tidak otomatis mengaktifkan akun. Sekretaris Cabang harus memeriksa dan mengaktifkan akun melalui Manajemen User.

Pendaftaran hanya tersedia ketika akun Cabang dan periode Cabang yang aktif sudah tersedia. Pengguna wajib memberikan nama dan alamat email yang benar serta tidak boleh mendaftarkan akun atas nama orang lain tanpa kewenangan.`,
  },
  {
    title: "4. Login dan Keamanan Akun",
    content: `Pengguna bertanggung jawab menjaga password dan perangkat yang dipakai untuk mengakses Laci Digital. Jangan membagikan password, kode OTP, cookie sesi, atau password sementara.

Login dengan Google hanya dapat dipakai oleh email yang sudah terdaftar dan akun yang sudah aktif. Fitur ini tidak membuat akun baru.

Sistem meminta lokasi browser saat login untuk mencatat keamanan aktivitas. Jika izin lokasi tidak diberikan, login tidak dapat dilanjutkan.

Jika akun diduga diakses pihak lain, segera hubungi Sekretaris Cabang untuk menonaktifkan akun atau mereset password. Reset password mengakhiri seluruh sesi lama akun tersebut.`,
  },
  {
    title: "5. Hak Akses dan Batas Kewenangan",
    content: `Pengguna hanya boleh membaca, menambah, mengubah, mengunduh, mengekspor, atau menghapus data yang berada dalam kewenangan perannya.

Sekretaris PAC tidak boleh mencoba mengakses data PAC lain, fungsi Cabang, backup, log email, atau manajemen akun.

Sekretaris Cabang dapat mengakses data lintas PAC hanya melalui fitur yang memang tersedia bagi peran Cabang. Hak akses Cabang harus digunakan untuk kepentingan administrasi dan pengawasan organisasi.`,
  },
  {
    title: "6. Data Organisasi dan Periode",
    content: `Data pada Laci Digital dikelompokkan menurut akun dan periode kepengurusan. Pengguna bertanggung jawab memilih periode yang benar sebelum memasukkan atau mengubah data.

Perubahan periode aktif memengaruhi data dan statistik yang ditampilkan. Fitur salin anggota membuat data baru pada periode tujuan dan tidak menghapus data sumber.

Menghapus periode dapat menghapus arsip, anggota, berkas, pengajuan, agenda, presensi, dan riwayat yang terhubung melalui relasi database. Periksa ringkasan serta konfirmasi sebelum menjalankan penghapusan.`,
  },
  {
    title: "7. Berkas dan Konten yang Diunggah",
    content: `Pengguna hanya boleh mengunggah dokumen dan foto yang sah, relevan dengan kegiatan organisasi, dan berada dalam kewenangannya.

Sistem memeriksa ukuran, ekstensi, dan isi file. Dokumen menerima format PDF, Word, atau PowerPoint sesuai batas yang ditampilkan pada form. Foto menerima format gambar yang didukung.

Pengguna dilarang mengunggah malware, dokumen palsu, konten yang melanggar hukum, atau data pribadi yang tidak diperlukan. Pengguna bertanggung jawab memastikan organisasi berhak menyimpan dan memproses berkas tersebut.`,
  },
  {
    title: "8. Pengajuan, Presensi, dan Notifikasi",
    content: `Sekretaris PAC bertanggung jawab atas isi dan file Pengajuan Berkas yang dikirim ke Cabang. Sekretaris Cabang dapat menerima atau menolak pengajuan berdasarkan pemeriksaan administrasi. Alasan penolakan harus digunakan untuk menjelaskan koreksi yang diperlukan.

Presensi hanya boleh dipakai untuk kegiatan organisasi yang sah. Pengurus bertanggung jawab menetapkan jadwal, membuka atau menutup sesi, dan menjaga QR atau tautan presensi dari penggunaan yang tidak semestinya.

Sistem dapat mengirim email untuk OTP, hasil verifikasi, pengajuan baru, dan perubahan status. Keterlambatan atau kegagalan email dapat terjadi jika layanan SMTP atau jaringan sedang bermasalah.`,
  },
  {
    title: "9. Larangan",
    content: `Pengguna dilarang:

• mencoba melewati autentikasi, pembatasan peran, atau kepemilikan data;
• mengganggu API, realtime, database, object storage, atau layanan email;
• menggunakan akun orang lain atau membagikan kredensial;
• memasukkan data palsu, memanipulasi presensi, atau mengubah status pengajuan tanpa kewenangan;
• menyebarkan hasil ekspor, backup, data anggota, atau berkas organisasi kepada pihak yang tidak berwenang;
• memakai Laci Digital untuk kepentingan di luar administrasi PC IPNU IPPNU Kabupaten Magetan.`,
  },
  {
    title: "10. Penghapusan dan Backup",
    content: `Penghapusan data dari aplikasi bersifat permanen. File terkait akan ikut dihapus dari object storage apabila proses penyimpanan tersedia.

Sekretaris Cabang dapat menghapus akun PAC. Relasi database membuat data yang terhubung dengan akun tersebut ikut terhapus.

Backup database baru dienkripsi dan sistem menyimpan hingga 10 backup terbaru. Karena itu, salinan data yang sudah dihapus dari data aktif dapat tetap berada dalam backup sampai backup tersebut terhapus oleh rotasi.`,
  },
  {
    title: "11. Ketersediaan dan Perubahan Layanan",
    content: `Laci Digital bergantung pada koneksi internet, database, object storage R2, layanan email, Google OAuth, reCAPTCHA, dan beberapa layanan data eksternal. Pengelola berusaha menjaga layanan tetap tersedia, tetapi tidak menjamin layanan bebas gangguan setiap saat.

Pengelola dapat melakukan pemeliharaan, memperbaiki keamanan, mengubah fitur, atau membatasi akses saat diperlukan. Perubahan ketentuan yang memengaruhi pengguna akan ditampilkan melalui sistem atau sarana komunikasi organisasi.`,
  },
  {
    title: "12. Pelanggaran dan Kontak",
    content: `Sekretaris Cabang dapat menonaktifkan atau menghapus akun PAC yang melanggar ketentuan, menggunakan data tanpa kewenangan, atau membahayakan sistem. Tindakan organisasi atau hukum lain dapat dilakukan sesuai tingkat pelanggaran.

Pertanyaan mengenai ketentuan ini dapat disampaikan melalui:

Email: lacipelajarnumagetan@gmail.com
WhatsApp: +62 858-5051-2135
Lokasi: Magetan, Jawa Timur, Indonesia`,
  },
];

export default function KetentuanClient({
  isAuthenticated = false,
}: {
  isAuthenticated?: boolean;
}) {
  return (
    <LegalPageShell
      badge="Legal"
      title="Ketentuan Penggunaan"
      description="Aturan ini menjelaskan siapa yang dapat menggunakan Laci Digital, batas akses setiap peran, serta tanggung jawab saat mengelola data organisasi."
      updatedAt="30 Oktober 2026"
      icon={FileText}
      sections={sections}
      isAuthenticated={isAuthenticated}
    />
  );
}
