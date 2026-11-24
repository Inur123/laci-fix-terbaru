"use client";

import { useState } from "react";
import {
  ArrowRight,
  ChevronDown,
  HelpCircle,
  LayoutGrid,
  Mail,
  MessageCircle,
  Shield,
  User,
  Wrench,
} from "lucide-react";

import { SiteFooter } from "@/components/features/public-v2/site-footer";
import { SiteHeader } from "@/components/features/public-v2/site-header";
import { cn } from "@/lib/utils";

const faqs = [
  {
    kategori: "Umum",
    items: [
      {
        q: "Apa itu Laci Digital?",
        a: "Laci Digital adalah sistem administrasi internal PC IPNU IPPNU Kabupaten Magetan. Sistem ini dipakai untuk mengelola periode kepengurusan, data anggota, arsip surat, berkas pimpinan, pengajuan PAC, agenda kegiatan, presensi, riwayat aktivitas, dan administrasi akun dalam satu tempat.",
      },
      {
        q: "Siapa yang dapat menggunakan Laci Digital?",
        a: "Pengguna utama Laci Digital adalah Sekretaris Cabang dan Sekretaris PAC yang memiliki akun aktif. Sekretaris Cabang mengelola data tingkat cabang serta administrasi akun PAC. Sekretaris PAC mengelola data milik PAC-nya dan mengirim pengajuan ke Cabang. Peserta kegiatan juga dapat mengisi presensi melalui tautan atau QR tanpa harus memiliki akun dashboard.",
      },
      {
        q: "Apakah Laci Digital dapat dibuka dari ponsel?",
        a: "Ya. Laci Digital dapat digunakan melalui browser di komputer, laptop, tablet, atau ponsel. Beberapa proses, seperti login, unggah berkas, scan QR, dan unduh hasil ekspor, tetap bergantung pada izin browser serta koneksi internet perangkat.",
      },
    ],
  },
  {
    kategori: "Akun & Registrasi",
    items: [
      {
        q: "Bagaimana alur pendaftaran akun PAC?",
        a: "Buka halaman Daftar, isi nama pimpinan, email, dan password, lalu setujui Ketentuan Penggunaan serta Kebijakan Privasi. Sistem membuat akun dengan peran Sekretaris PAC dalam keadaan belum aktif. Kode OTP 6 digit dikirim ke email dan berlaku selama 5 menit. Setelah email terverifikasi, Sekretaris Cabang tetap harus mengaktifkan akun melalui menu Manajemen User sebelum akun dapat login.",
      },
      {
        q: "Mengapa halaman pendaftaran dapat ditutup sementara?",
        a: "Pendaftaran PAC hanya dibuka ketika akun Sekretaris Cabang aktif dan Cabang memiliki periode kepengurusan aktif. Jika syarat ini belum terpenuhi, halaman pendaftaran menampilkan pemberitahuan agar pengguna menghubungi Sekretaris Cabang.",
      },
      {
        q: "Apakah login dengan Google dapat membuat akun baru?",
        a: "Tidak. Login Google hanya bekerja untuk alamat email yang sudah terdaftar di Laci Digital dan akunnya sudah diaktifkan. Google digunakan untuk mencocokkan identitas email, bukan untuk mendaftarkan akun PAC baru.",
      },
      {
        q: "Mengapa login meminta akses lokasi?",
        a: "Lokasi digunakan sebagai bagian dari catatan keamanan aktivitas. Browser menyimpan koordinat dan alamat ringkas selama maksimal 24 jam, lalu backend mencatat lokasi bersama IP, browser, dan jenis perangkat pada aktivitas yang relevan. Jika izin lokasi ditolak, proses login tidak dapat dilanjutkan.",
      },
      {
        q: "Saya lupa password. Bagaimana cara meresetnya?",
        a: "Hubungi Sekretaris Cabang. Melalui menu Manajemen User, Sekretaris Cabang dapat membuat password sementara untuk akun PAC. Proses reset mengakhiri seluruh sesi lama pada akun tersebut. Setelah berhasil login, pengguna dapat mengganti password dari halaman Profil.",
      },
      {
        q: "Mengapa akun yang sudah verifikasi email masih belum bisa login?",
        a: "Verifikasi email dan aktivasi akun adalah dua tahap yang berbeda. OTP memastikan alamat email dapat diakses, sedangkan aktivasi akun dilakukan oleh Sekretaris Cabang. Login baru dapat dilakukan setelah kedua tahap selesai.",
      },
    ],
  },
  {
    kategori: "Fitur Dashboard",
    items: [
      {
        q: "Apa perbedaan menu untuk Sekretaris Cabang dan Sekretaris PAC?",
        a: "Kedua peran memiliki Dashboard, Pengajuan Berkas, Arsip Surat, Berkas Pimpinan, Data Anggota, Presensi, Periode, Profil, dan Riwayat Aktivitas. Sekretaris PAC juga memiliki Referensi Pengajuan. Sekretaris Cabang memiliki Berkas SP, Agenda Kegiatan, Manajemen User, Log Email, dan Backup Database. Data serta aksi tertentu tetap dibatasi oleh kepemilikan dan peran di backend.",
      },
      {
        q: "Bagaimana periode kepengurusan memengaruhi data?",
        a: "Arsip, anggota, berkas, pengajuan, agenda, presensi, statistik, dan riwayat aktivitas dibaca dalam konteks periode. Pengguna dapat membuat periode, mengaktifkan satu periode, melihat periode lain, dan memindahkan salinan anggota ke periode tujuan. Menghapus periode juga menghapus data yang terhubung dengannya, sehingga konfirmasi penghapusan harus diperiksa dengan teliti.",
      },
      {
        q: "Bagaimana alur Pengajuan Berkas dari PAC ke Cabang?",
        a: "Sekretaris PAC membuat pengajuan dengan nomor surat, penerima, tanggal, keperluan, deskripsi, dan file. Sekretaris Cabang meninjau pengajuan lalu menerima atau menolaknya. Penolakan memuat alasan. Perubahan status dicatat dan sistem mengirim email notifikasi jika layanan email tersedia. PAC dapat memantau status dan melihat pengajuan yang diterima melalui Referensi Pengajuan.",
      },
      {
        q: "Apa perbedaan Arsip Surat, Berkas SP, dan Berkas Pimpinan?",
        a: "Arsip Surat menyimpan surat masuk atau keluar dan file bersifat opsional. Berkas Pimpinan menyimpan dokumen kepengurusan Cabang atau PAC dan file juga dapat bersifat opsional sesuai form. Berkas SP hanya tersedia untuk Sekretaris Cabang dan file wajib diunggah.",
      },
      {
        q: "Bagaimana Data Anggota dikelola?",
        a: "Data Anggota menyimpan identitas, kontak, jabatan, pendidikan, perkaderan, dan foto. Daftar dapat dicari, difilter, diurutkan, dipaginasi, dan diekspor. Fitur impor Excel tidak digunakan. Sekretaris Cabang dapat melihat data lintas akun sesuai kewenangannya, sedangkan Sekretaris PAC dibatasi pada data miliknya.",
      },
      {
        q: "Bagaimana agenda kegiatan dan presensi bekerja?",
        a: "Agenda Kegiatan dikelola oleh Sekretaris Cabang. Presensi dapat dibuat oleh Cabang maupun PAC dan menghasilkan tautan atau QR untuk peserta. Form presensi hanya menerima data saat sesi berada dalam jadwal yang diizinkan, kecuali pengurus membuka sesi secara manual. Pengurus juga dapat menutup sesi manual dan melihat detail peserta.",
      },
      {
        q: "Apakah data dapat diekspor?",
        a: "Ya. Modul yang menyediakan tombol Export dapat mengunduh data sesuai filter yang sedang dipakai. Sistem mengambil seluruh halaman data yang cocok sebelum membuat file agar hasil ekspor tidak terpotong. Fitur impor Excel sudah tidak digunakan.",
      },
      {
        q: "Apa fungsi Riwayat Aktivitas, Log Email, dan Backup Database?",
        a: "Riwayat Aktivitas mencatat aksi penting seperti login, perubahan data, unggah, penghapusan, dan perubahan status. Pengguna PAC melihat riwayat miliknya, sedangkan Cabang dapat memantau riwayat yang lebih luas. Log Email dan Backup Database hanya tersedia untuk Cabang. Backup baru dienkripsi dan sistem merotasi penyimpanan hingga 10 backup terbaru.",
      },
    ],
  },
  {
    kategori: "Keamanan & Privasi",
    items: [
      {
        q: "Bagaimana Laci Digital melindungi akun dan data?",
        a: "Password disimpan dalam bentuk hash, token sesi disimpan dalam bentuk digest, dan cookie sesi menggunakan HttpOnly serta SameSite. Data teks sensitif dan file baru dienkripsi dengan AES-GCM. Backend juga menerapkan pemeriksaan peran, kepemilikan data, pembatasan ukuran request, rate limit pada endpoint sensitif, validasi origin, dan token unduhan berjangka pendek.",
      },
      {
        q: "Data keamanan apa yang dicatat?",
        a: "Sesi dan riwayat aktivitas dapat mencatat waktu, alamat IP, user agent, nama browser, jenis perangkat, dan lokasi ringkas yang diberikan browser saat login. Catatan ini digunakan untuk audit internal dan penelusuran aktivitas akun.",
      },
      {
        q: "Apakah pengguna PAC dapat melihat data PAC lain?",
        a: "Tidak untuk data yang dibatasi kepemilikan. Backend memeriksa akun pemilik, periode, dan peran sebelum membaca atau mengubah data. Sekretaris Cabang memiliki akses lintas PAC hanya pada fitur yang memang diberikan kepada peran Cabang.",
      },
      {
        q: "Apa yang terjadi saat data atau akun dihapus?",
        a: "Penghapusan dari aplikasi bersifat permanen. Data utama di database dihapus dan file terkait di object storage ikut dijadwalkan untuk dihapus. Jika akun PAC dihapus oleh Cabang, data yang memiliki relasi ke akun tersebut ikut terhapus melalui aturan database. Salinan pada backup dapat tetap ada sampai terhapus oleh rotasi backup.",
      },
    ],
  },
  {
    kategori: "Bantuan Teknis",
    items: [
      {
        q: "Kode OTP tidak masuk. Apa yang harus dilakukan?",
        a: "Periksa folder Spam atau Promosi, pastikan alamat email benar, lalu gunakan tombol Kirim ulang setelah waktu tunggu selesai. Setiap pengiriman membuat kode baru dan membatalkan kode sebelumnya. Jika layanan email gagal, hubungi Sekretaris Cabang atau tim pengelola.",
      },
      {
        q: "Mengapa unggah file ditolak?",
        a: "Sistem memeriksa ekstensi, isi file, dan ukuran. Dokumen menerima PDF, Word, atau PowerPoint sesuai batas pada form. Foto menerima format gambar yang didukung. Ubah file ke format yang diperbolehkan dan pastikan ukurannya tidak melebihi batas yang ditampilkan.",
      },
      {
        q: "Halaman lambat atau data belum berubah. Apa yang harus dilakukan?",
        a: "Periksa koneksi internet lalu muat ulang halaman. Laci Digital memakai pembaruan realtime untuk beberapa perubahan, tetapi koneksi realtime dapat terputus sementara. Jika masalah tetap terjadi, catat menu, waktu kejadian, dan pesan error, lalu kirimkan melalui tombol Laporkan Bug atau kontak pengelola.",
      },
      {
        q: "Bagaimana menghubungi pengelola?",
        a: "Kirim email ke lacipelajarnumagetan@gmail.com atau WhatsApp ke +62 858-5051-2135. Sertakan nama menu, waktu kejadian, dan tangkapan layar jika ada. Jangan mengirim password, kode OTP, atau cookie sesi.",
      },
    ],
  },
];

function AccordionItem({
  itemId,
  q,
  a,
  isOpen,
  onToggle,
}: {
  itemId: string;
  q: string;
  a: string;
  isOpen: boolean;
  onToggle: () => void;
}) {
  return (
    <div
      className={cn(
        "rounded-2xl border transition-colors",
        isOpen
          ? "border-slate-200/80 bg-[#f0fdf4]"
          : "border-slate-200/80 bg-white hover:border-slate-300",
      )}
    >
      <button
        id={`${itemId}-trigger`}
        type="button"
        onClick={onToggle}
        className="flex min-h-12 w-full cursor-pointer items-center justify-between gap-4 rounded-2xl p-5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#0a8744] focus-visible:ring-offset-2"
        aria-expanded={isOpen}
        aria-controls={`${itemId}-content`}
      >
        <span className="text-sm font-semibold leading-snug text-slate-900">
          {q}
        </span>
        <ChevronDown
          aria-hidden="true"
          className={cn(
            "h-5 w-5 shrink-0 text-slate-500 transition-transform duration-200",
            isOpen && "rotate-180 text-[#15803d]",
          )}
        />
      </button>
      <div
        id={`${itemId}-content`}
        role="region"
        aria-labelledby={`${itemId}-trigger`}
        className={cn(
          "overflow-hidden transition-[max-height] duration-200",
          isOpen ? "max-h-[1200px]" : "max-h-0",
        )}
      >
        <p className="whitespace-pre-line px-5 pb-5 text-sm leading-7 text-slate-600">
          {a}
        </p>
      </div>
    </div>
  );
}
const categoryAppearance = {
  Umum: {
    icon: MessageCircle,
    description: "Gambaran dasar dan pengguna Laci Digital.",
    color: "bg-[#e6f7ed] text-[#0a8744]",
  },
  "Akun & Registrasi": {
    icon: User,
    description: "Pendaftaran, OTP, aktivasi, dan login.",
    color: "bg-[#f3e8ff] text-[#7c3aed]",
  },
  "Fitur Dashboard": {
    icon: LayoutGrid,
    description: "Menu, alur kerja, periode, dan ekspor data.",
    color: "bg-[#ffedd5] text-[#ea580c]",
  },
  "Keamanan & Privasi": {
    icon: Shield,
    description: "Batas akses, enkripsi, dan penghapusan data.",
    color: "bg-[#e0f2fe] text-[#0284c7]",
  },
  "Bantuan Teknis": {
    icon: Wrench,
    description: "Solusi untuk kendala yang sering terjadi.",
    color: "bg-[#e6f7ed] text-[#0a8744]",
  },
} as const;

export default function FAQClient({
  isAuthenticated = false,
}: {
  isAuthenticated?: boolean;
}) {
  const [openIndex, setOpenIndex] = useState<string | null>(null);

  return (
    <div className="min-h-screen overflow-x-clip bg-[#fbfdfc] text-[#091526]">
      <div className="relative overflow-hidden bg-gradient-to-br from-[#e3f6eb] via-[#f3fbf6] to-[#dff4e7]">
        <div
          aria-hidden="true"
          className="absolute -right-20 top-12 size-72 rounded-full bg-[#bfe7cf]/45 blur-3xl"
        />
        <div
          aria-hidden="true"
          className="absolute -bottom-20 left-10 size-60 rounded-full bg-white/65 blur-2xl"
        />
        <SiteHeader isAuthenticated={isAuthenticated} />
        <div className="relative mx-auto max-w-4xl px-4 pb-16 pt-10 text-center sm:px-6 sm:pb-20 sm:pt-14 lg:px-8">
          <div className="mx-auto flex size-12 items-center justify-center rounded-2xl bg-white/85 text-[#0a8744] shadow-sm ring-1 ring-emerald-100">
            <HelpCircle className="size-6" />
          </div>
          <p className="mt-5 text-xs font-extrabold uppercase tracking-[0.2em] text-[#0a8744]">
            Pusat Bantuan
          </p>
          <h1 className="mt-3 text-3xl font-extrabold tracking-tight sm:text-5xl">
            Pertanyaan yang sering ditanyakan
          </h1>
          <p className="mx-auto mt-4 max-w-2xl text-sm leading-7 text-slate-600 sm:text-base">
            Jawaban berdasarkan fitur dan aturan yang saat ini berlaku di Laci
            Digital.
          </p>
        </div>
      </div>

      <main className="mx-auto max-w-4xl px-4 py-12 sm:px-6 sm:py-16 lg:px-8">
        <div className="space-y-12">
          {faqs.map((section) => {
            const appearance =
              categoryAppearance[
                section.kategori as keyof typeof categoryAppearance
              ];
            const Icon = appearance.icon;

            return (
              <section key={section.kategori}>
                <div className="mb-5 flex items-center gap-3">
                  <div
                    className={cn(
                      "flex size-11 shrink-0 items-center justify-center rounded-2xl",
                      appearance.color,
                    )}
                  >
                    <Icon className="size-5" />
                  </div>
                  <div>
                    <h2 className="text-sm font-extrabold uppercase tracking-[0.13em] text-[#091526]">
                      {section.kategori}
                    </h2>
                    <p className="mt-0.5 text-sm text-slate-500">
                      {appearance.description}
                    </p>
                  </div>
                </div>
                <div className="space-y-3">
                  {section.items.map((item, index) => {
                    const key = `${section.kategori}-${index}`;
                    return (
                      <AccordionItem
                        key={key}
                        itemId={key.replace(/[^a-zA-Z0-9-]/g, "-")}
                        q={item.q}
                        a={item.a}
                        isOpen={openIndex === key}
                        onToggle={() =>
                          setOpenIndex(openIndex === key ? null : key)
                        }
                      />
                    );
                  })}
                </div>
              </section>
            );
          })}
        </div>

        <section className="mt-16 rounded-[28px] bg-gradient-to-br from-[#075d32] to-[#043c22] px-6 py-9 text-center text-white shadow-xl shadow-emerald-950/10 sm:px-10 sm:py-11">
          <div className="mx-auto flex size-12 items-center justify-center rounded-2xl bg-white/10 text-emerald-100 ring-1 ring-white/15">
            <Mail className="size-5" />
          </div>
          <h2 className="mt-4 text-2xl font-extrabold">
            Masih memerlukan bantuan?
          </h2>
          <p className="mx-auto mt-2 max-w-xl text-sm leading-6 text-emerald-100/85">
            Jelaskan menu, waktu kejadian, dan pesan error agar tim pengelola
            dapat memeriksa masalah dengan tepat.
          </p>
          <a
            href="mailto:lacipelajarnumagetan@gmail.com"
            className="mt-6 inline-flex min-h-11 items-center justify-center gap-2 rounded-xl bg-white px-5 py-2.5 text-sm font-bold text-[#075d32] transition-colors hover:bg-emerald-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-offset-2 focus-visible:ring-offset-[#075d32]"
          >
            Kirim email
            <ArrowRight className="size-4" />
          </a>
        </section>
      </main>
      <SiteFooter />
    </div>
  );
}
