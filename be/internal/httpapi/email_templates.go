package httpapi

import (
	"bytes"
	"html/template"
	"time"
)

const emailLayout = `<!DOCTYPE html>
<html lang="id"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>{{.Title}}</title></head>
<body style="font-family:Arial,sans-serif;line-height:1.6;color:#333;max-width:600px;margin:0 auto;padding:20px;background:#fff">
<div style="display:none;max-height:0;overflow:hidden">{{.Preheader}}</div>
<div style="background:{{.Color}};padding:30px;text-align:center;border-radius:10px 10px 0 0">
<h1 style="color:#fff;margin:0;font-size:28px">Laci Digital</h1><p style="color:#f0f0f0;margin:10px 0 0">PC IPNU IPPNU Magetan</p></div>
<div style="background:#fff;padding:30px;border:1px solid #e0e0e0;border-top:none;border-radius:0 0 10px 10px">
<h2 style="color:{{.Color}};margin-top:0">{{.Heading}}</h2>{{.Body}}
<div style="margin-top:40px;padding-top:20px;border-top:1px dashed #e5e7eb;text-align:center;color:#6b7280;font-size:12px">
<p style="margin:0 0 5px 0">&copy; {{.Year}} <strong style="color:#15803d">Laci Digital</strong>. Semua hak dilindungi.</p>
<p style="margin:0">PC IPNU IPPNU Magetan | Magetan, Jawa Timur, Indonesia</p></div></div></body></html>`

func renderEmail(title, preheader, heading, color, body string, data any) string {
	var content bytes.Buffer
	if err := template.Must(template.New("body").Parse(body)).Execute(&content, data); err != nil {
		return ""
	}
	view := struct {
		Title, Preheader, Heading, Color string
		Body                             template.HTML
		Year                             int
	}{title, preheader, heading, color, template.HTML(content.String()), time.Now().Year()}
	var result bytes.Buffer
	if err := template.Must(template.New("layout").Parse(emailLayout)).Execute(&result, view); err != nil {
		return ""
	}
	return result.String()
}

func verificationEmailHTML(name, otp string) string {
	return renderEmail("Kode Verifikasi Email", "Kode verifikasi email Laci Digital Anda.", "Verifikasi Email Anda", "#15803d", `
<p>Halo <strong>{{.Name}}</strong>,</p>
<p>Terima kasih telah mendaftar di Laci Digital. Gunakan kode OTP di bawah ini untuk melengkapi proses verifikasi email Anda:</p>
<div style="text-align:center;margin:40px 0"><div style="display:inline-block;background:#f0fdf4;border:2px dashed #15803d;padding:20px 40px;border-radius:12px">
<span style="font-family:monospace;font-size:42px;font-weight:bold;color:#15803d;letter-spacing:10px">{{.OTP}}</span></div></div>
<p style="color:#ef4444;font-size:14px;text-align:center"><strong>Penting:</strong> Kode ini hanya berlaku selama 5 menit.</p>
<p style="color:#718096;font-size:14px">Jika Anda tidak merasa mendaftar di Laci Digital, harap abaikan email ini. Jangan berikan kode ini kepada siapapun.</p>`, struct{ Name, OTP string }{name, otp})
}

func verifiedSuccessEmailHTML(name string) string {
	return renderEmail("Email Berhasil Diverifikasi", "Alamat email Anda berhasil diverifikasi dan menunggu aktivasi.", "Akun Berhasil Diverifikasi!", "#15803d", `
<p>Halo <strong>{{.}}</strong>,</p><p>Selamat! Alamat email Anda telah berhasil diverifikasi.</p>
<p>Akun Laci Digital Anda saat ini sedang menunggu aktivasi oleh <strong>Sekretaris Cabang</strong> PC IPNU IPPNU Magetan sebelum Anda dapat mulai mengelola berkas dan data organisasi secara penuh.</p>
<p style="color:#666;font-size:14px;margin-top:30px">Jika Anda mengalami kendala saat menggunakan aplikasi, jangan ragu untuk menghubungi tim support kami.</p>`, name)
}

type pengajuanEmailData struct {
	UserName, PACName, Number, Date, DetailURL, Status, Reason string
}

func pengajuanCreatedUserEmailHTML(data pengajuanEmailData) string {
	return renderEmail("Pengajuan Berhasil", "Pengajuan PAC Anda berhasil dikirim.", "Pengajuan Telah Diterima", "#15803d", `
<p>Halo <strong>{{.UserName}}</strong>,</p><p>Terima kasih telah melakukan pengajuan data untuk <strong>{{.PACName}}</strong> melalui sistem Laci Digital.</p>
<div style="background:#f9fafb;padding:20px;border-radius:8px;margin:20px 0;border:1px solid #e5e7eb">
<p style="margin:0;color:#6b7280">Ringkasan Pengajuan:</p><p><strong>PAC:</strong> {{.PACName}}</p><p><strong>Tanggal:</strong> {{.Date}}</p>
<p><strong>Status:</strong> <span style="color:#d97706;font-weight:bold">Menunggu Verifikasi</span></p></div>
<p>Tim admin PC IPNU IPPNU Magetan akan segera meninjau pengajuan Anda. Anda dapat memantau status pengajuan melalui dashboard.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.DetailURL}}" style="background:#15803d;color:#fff;padding:14px 30px;text-decoration:none;border-radius:8px;font-weight:bold">Lihat Detail Pengajuan</a></p>`, data)
}

func pengajuanCreatedAdminEmailHTML(data pengajuanEmailData) string {
	return renderEmail("Notifikasi Pengajuan Baru", "Ada pengajuan PAC baru yang perlu ditinjau.", "Pengajuan Baru Perlu Ditinjau", "#1e293b", `
<p>Halo Admin,</p><p>Sistem mendeteksi adanya pengajuan baru yang dikirimkan oleh user. Mohon segera lakukan tindak lanjut.</p>
<div style="background:#f1f5f9;padding:20px;border-radius:8px;margin:20px 0;border:1px solid #e2e8f0">
<p style="margin:0;color:#64748b">Informasi Pengajuan:</p><p><strong>Pengaju:</strong> {{.UserName}}</p>
<p><strong>PAC:</strong> {{.PACName}}</p><p><strong>Tanggal:</strong> {{.Date}}</p></div>
<p>Silakan masuk ke panel admin untuk meninjau berkas yang diajukan.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.DetailURL}}" style="background:#0f172a;color:#fff;padding:14px 30px;text-decoration:none;border-radius:8px;font-weight:bold">Tinjau Pengajuan Sekarang</a></p>`, data)
}

func pengajuanStatusEmailHTML(data pengajuanEmailData) string {
	accepted := data.Status == "DITERIMA"
	color, label := "#ef4444", "Ditolak"
	if accepted {
		color, label = "#15803d", "Diterima"
	}
	message := "Silakan lakukan revisi sesuai alasan di atas dan ajukan kembali jika diperlukan. Jika ada pertanyaan, silakan hubungi Sekretaris Cabang."
	if accepted {
		message = "Terima kasih atas kerja samanya. Pengajuan Anda telah resmi tercatat di sistem."
	}
	return renderEmail("Update Status Pengajuan", "Status pengajuan PAC Anda telah diperbarui.", "Pengajuan Anda Telah "+label, color, `
<p>Halo <strong>{{.UserName}}</strong>,</p>
<p>Kami ingin menginformasikan bahwa pengajuan PAC Anda untuk <strong>{{.PACName}}</strong> telah selesai ditinjau oleh Sekretaris Cabang.</p>
<div style="background:#f9fafb;padding:20px;border-radius:8px;margin:20px 0;border:1px solid #e5e7eb">
<p style="margin:0;color:#6b7280">Informasi Pengajuan:</p><p><strong>PAC:</strong> {{.PACName}}</p>
<p><strong>Status:</strong> {{.StatusLabel}}</p>{{if .Reason}}<p><strong>Alasan Penolakan:</strong> {{.Reason}}</p>{{end}}</div>
<p>{{.Message}}</p><p style="text-align:center;margin:30px 0"><a href="{{.DetailURL}}" style="background:{{.Color}};color:#fff;padding:14px 30px;text-decoration:none;border-radius:8px;font-weight:bold">Lihat Detail Di Dashboard</a></p>`, struct {
		pengajuanEmailData
		StatusLabel, Message, Color string
	}{data, label, message, color})
}
