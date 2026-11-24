# Git Workflow & Release Guidelines

Dokumen ini adalah panduan standar dan instruksi wajib (baik untuk developer manusia maupun AI Assistant seperti Cursor, Claude, Antigravity, ChatGPT, dll.) setiap kali melakukan commit dan push ke repository **laci-fix-terbaru**.

---

## 1. Prinsip Utama: Jangan Push "Polosan"
Setiap commit dan push harus:
1. Mengikuti format **Conventional Commits**.
2. Memiliki deskripsi perubahan yang jelas dan berstruktur.
3. Memperbarui versi pada `fe/package.json` dan `CHANGELOG.md` bila ada rilis/milestone baru.
4. Membuat **Git Annotated Tag** (misal `v1.1.0`) untuk setiap rilis versi baru.

---

## 2. Format Commit (Conventional Commits)

Format standar:
```
<type>(<scope>): <ringkasan singkat perubahan dalam bahasa Indonesia/Inggris> [vX.Y.Z]

- Detail perubahan 1
- Detail perubahan 2
- Verifikasi yang telah dilakukan
```

### Jenis `<type>` yang Digunakan:
- `feat`: Penambahan fitur baru (misal: halaman baru, integrasi API baru).
- `fix`: Perbaikan bug atau penyesuaian UI/styling.
- `perf`: Optimasi performa dan efisiensi query / rendering.
- `refactor`: Restrukturisasi kode tanpa mengubah fungsionalitas.
- `style`: Penyesuaian formatting, whitespace, atau CSS minor.
- `docs`: Pembaruan dokumentasi (README, CHANGELOG, API docs).
- `chore`: Update dependency, konfigurasi build, atau skrip repo.

### Contoh `<scope>`:
`agenda`, `arsip`, `pengajuan`, `anggota`, `presensi`, `user`, `ui`, `auth`, `api`, `db`.

---

## 3. Alur Rilis & Push (Semantic Versioning)

Ikuti kaidah **Semantic Versioning** (`vMAJOR.MINOR.PATCH`):
- **PATCH** (`v1.1.0` -> `v1.1.1`): Perbaikan bug, styling minor, perbaikan alignment.
- **MINOR** (`v1.1.0` -> `v1.2.0`): Fitur baru, halaman baru, penambahan skema yang backwards-compatible.
- **MAJOR** (`v1.0.0` -> `v2.0.0`): Perubahan arsitektur besar / breaking change.

### Langkah-langkah Wajib Sebelum & Saat Push:
1. **Lakukan Pemeriksaan Kualitas**:
   ```bash
   npm --prefix fe run typecheck
   cd be && go test ./...
   ```
2. **Update Versi**:
   - Di `fe/package.json`: perbarui field `"version"`.
   - Di `CHANGELOG.md`: catat riwayat perubahan di bawah judul `## [vX.Y.Z] - YYYY-MM-DD`.
3. **Stage & Commit**:
   ```bash
   git add .
   git commit -m "feat(ui): implement responsive touch targets and search icon alignment [v1.1.0]

   - Harmonize form submit buttons to upload/cancel icon across all forms
   - Standardize search icon positioning with vertical center
   - Add national holiday calendar integration on agenda kegiatan"
   ```
4. **Buat Git Tag**:
   ```bash
   git tag -a v1.1.0 -m "Release v1.1.0: UI & mobile responsiveness improvements, holiday calendar, and search alignment"
   ```
5. **Push ke GitHub Beserta Tags**:
   ```bash
   git push origin main --tags
   ```

---

## 4. Skrip Otomatis
Anda juga dapat menjalankan skrip helper release yang telah disediakan:
```bash
./scripts/release.sh <patch|minor|major> "Pesan ringkasan rilis"
```
Skrip ini akan otomatis memvalidasi typecheck, menaikkan versi, membuat commit, membuat tag, dan melakukan push ke origin.
