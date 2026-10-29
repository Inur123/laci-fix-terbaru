package httpapi

func (a *API) dataOpenAPISpec() map[string]any {
	query := func(name, description, kind string, required bool) map[string]any {
		return map[string]any{
			"name": name, "in": "query", "required": required,
			"description": description, "schema": map[string]any{"type": kind},
		}
	}
	security := []map[string]any{{"sessionCookie": []string{}}}
	arsipList := map[string]any{
		"summary":     "Daftar arsip milik pengguna pada periode yang dipilih",
		"description": "Filter jenis dan organisasi berlaku sebelum pengurutan dan pagination. q mencari teks setelah dekripsi sehingga lebih mahal untuk kumpulan data besar. Data akun lain tidak dapat dipilih hanya dengan email atau userId.",
		"security":    security,
		"parameters": []map[string]any{
			query("jenis", "MASUK, KELUAR, atau ALL", "string", false),
			query("organisasi", "IPNU, IPPNU, BERSAMA, CBP_KPP, atau ALL", "string", false),
			query("q", "Pencarian isi surat yang terenkripsi", "string", false),
			query("sortKey", "createdAt (default), tanggal, updatedAt, organisasi, jenisSurat, noSurat, pengirimPenerima, atau perihal", "string", false),
			query("sortDir", "asc atau desc; default desc", "string", false),
			query("page", "Nomor halaman mulai dari 1", "integer", false),
			query("limit", "Jumlah per halaman, maksimal 500", "integer", false),
		},
		"responses": map[string]any{
			"200": map[string]any{"description": "Daftar arsip dan metadata pagination", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{
				"type": "object", "required": []string{"data", "total", "totalPages"},
				"properties": map[string]any{
					"data":  map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/ArsipSurat"}},
					"total": map[string]any{"type": "integer"}, "totalPages": map[string]any{"type": "integer"},
				},
			}}}},
			"400": map[string]any{"description": "Filter tidak valid"},
			"401": map[string]any{"description": "Sesi tidak valid"},
		},
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title": "Laci Digital Data API", "version": "2.0.0",
			"description": "Spesifikasi ini saat ini mencakup endpoint baca arsip yang sudah diverifikasi. Endpoint lain belum didokumentasikan di sini; daftar ini bukan klaim bahwa API lain tidak tersedia.",
		},
		"servers": []map[string]string{{"url": "/"}},
		"components": map[string]any{
			"securitySchemes": map[string]any{"sessionCookie": map[string]any{"type": "apiKey", "in": "cookie", "name": a.cfg.SessionCookie}},
			"schemas": map[string]any{"ArsipSurat": map[string]any{
				"type": "object", "description": "Metadata arsip; field teks dikembalikan sebagai teks hasil dekripsi kepada pemilik yang terautentikasi.",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"}, "userId": map[string]any{"type": "string"}, "periodeId": map[string]any{"type": "string"},
					"noSurat": map[string]any{"type": "string"}, "jenisSurat": map[string]any{"type": "string", "enum": []string{"MASUK", "KELUAR"}},
					"organisasi": map[string]any{"type": "string", "nullable": true}, "tanggal": map[string]any{"type": "string", "format": "date-time"},
					"pengirimPenerima": map[string]any{"type": "string"}, "perihal": map[string]any{"type": "string"},
					"deskripsi": map[string]any{"type": "string", "nullable": true}, "file": map[string]any{"type": "string", "nullable": true, "description": "Kunci penyimpanan, bukan URL unduhan publik"},
				},
			}},
		},
		"paths": map[string]any{
			"/api/arsip": map[string]any{"get": arsipList},
			"/api/arsip/{id}": map[string]any{"get": map[string]any{
				"summary": "Detail arsip milik pengguna", "security": security,
				"parameters": []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}}},
				"responses":  map[string]any{"200": map[string]any{"description": "Detail arsip"}, "401": map[string]any{"description": "Sesi tidak valid"}, "404": map[string]any{"description": "Arsip tidak ditemukan"}},
			}},
		},
	}
}
