package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"laci-backend/internal/domain"
)

func (a *API) StartRealtimeListener(ctx context.Context) {
	go a.publishNotifications(ctx)
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			conn, err := a.db.Acquire(ctx)
			if err != nil {
				if !waitOrDone(ctx, 3*time.Second) {
					return
				}
				continue
			}
			_, err = conn.Exec(ctx, "LISTEN laci_realtime")
			if err != nil {
				conn.Release()
				if !waitOrDone(ctx, 3*time.Second) {
					return
				}
				continue
			}
			for {
				notification, err := conn.Conn().WaitForNotification(ctx)
				if err != nil {
					break
				}
				if notificationOrigin(notification.Payload) != a.instanceID {
					a.hub.publish(notification.Payload)
				}
			}
			conn.Release()
			if !waitOrDone(ctx, time.Second) {
				return
			}
		}
	}()
}

func waitOrDone(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func notificationOrigin(value string) string {
	var event struct {
		Origin string `json:"_origin"`
	}
	_ = json.Unmarshal([]byte(value), &event)
	return event.Origin
}

func (a *API) publishNotifications(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-a.notifyQueue:
			queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, err := a.db.Exec(queryCtx, `SELECT pg_notify('laci_realtime',$1)`, payload)
			cancel()
			if err != nil && ctx.Err() == nil {
				slog.Warn("realtime publish failed", "error", err)
			}
		}
	}
}

func (a *API) notify(_ context.Context, payload map[string]any) {
	payload["_origin"] = a.instanceID
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	value := string(encoded)
	a.hub.publish(value)
	select {
	case a.notifyQueue <- value:
	default:
		slog.Warn("realtime queue full; remote notification dropped")
	}
	if model := mutationModel(payload); model != "" && payload["type"] != "mutation" {
		a.notifyMutation(model, payload)
	}
}

func mutationModel(payload map[string]any) string {
	if model, ok := payload["model"].(string); ok && model != "" {
		return model
	}
	kind, _ := payload["type"].(string)
	if kind == "log" && payload["module"] == "PRESENSI" {
		return "Presensi"
	}
	switch {
	case strings.HasPrefix(kind, "arsip."):
		return "ArsipSurat"
	case strings.HasPrefix(kind, "berkas_pimpinan."):
		return "BerkasPimpinan"
	case strings.HasPrefix(kind, "berkas_sp."):
		return "BerkasSP"
	case strings.HasPrefix(kind, "anggota."):
		return "Anggota"
	case strings.HasPrefix(kind, "agenda."):
		return "AgendaKegiatan"
	case strings.HasPrefix(kind, "pengajuan."):
		return "PengajuanBerkas"
	case strings.HasPrefix(kind, "periode."):
		return "Periode"
	}
	return ""
}

func (a *API) notifyMutation(model string, original map[string]any) {
	payload := map[string]any{"type": "mutation", "model": model, "_origin": a.instanceID}
	if id, ok := original["id"].(string); ok {
		payload["id"] = id
	}
	if userID, ok := original["userId"].(string); ok {
		payload["userId"] = userID
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	value := string(encoded)
	a.hub.publish(value)
	select {
	case a.notifyQueue <- value:
	default:
		slog.Warn("realtime queue full; remote mutation dropped", "model", model)
	}
}

func (a *API) realtime(w http.ResponseWriter, r *http.Request) {
	user, _, err := a.currentUser(r)
	if err != nil || !user.IsActive {
		user = domain.User{}
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusNotImplemented, "STREAM_UNSUPPORTED", "Streaming tidak didukung")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	ch := make(chan string, 16)
	a.hub.add(ch)
	defer a.hub.remove(ch)
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.streamDone:
			return
		case <-r.Context().Done():
			return
		case event := <-ch:
			visible, ok := visibleRealtimeEvent(event, user)
			if !ok {
				continue
			}
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := fmt.Fprintf(w, "data: %s\n\n", visible); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func visibleRealtimeEvent(raw string, user domain.User) (string, bool) {
	var event map[string]any
	if json.Unmarshal([]byte(raw), &event) != nil {
		return "", false
	}
	delete(event, "_origin")
	kind, _ := event["type"].(string)
	model, _ := event["model"].(string)
	publicPresensi := (kind == "log" && event["module"] == "PRESENSI") || (kind == "mutation" && (model == "Presensi" || model == "PresensiData"))
	if user.ID == "" && !publicPresensi {
		return "", false
	}
	if user.ID == "" {
		for key := range event {
			switch key {
			case "type", "model", "module", "entityId", "presensiId":
			default:
				delete(event, key)
			}
		}
	}
	if user.Role != "SEKRETARIS_CABANG" && !publicPresensi {
		owner, _ := event["userId"].(string)
		id, _ := event["id"].(string)
		if owner != user.ID && id != user.ID {
			for _, key := range []string{"id", "userId", "entityId", "presensiId", "status", "action"} {
				delete(event, key)
			}
		}
	}
	encoded, err := json.Marshal(event)
	return string(encoded), err == nil
}

func (a *API) createLog(ctx context.Context, r *http.Request, user domain.User, action, module, description, entityID string) error {
	periodeID, err := a.targetPeriode(ctx, user.ID, "")
	if err != nil {
		return nil
	}
	location := requestLocation(r)
	_, err = a.db.Exec(ctx, `INSERT INTO "LogActivity"("id","userId","periodeId","action","module","description","entityId","ipAddress","userAgent","browser","device","location","createdAt") VALUES($1,$2,$3,$4::"LogAction",$5::"LogModule",$6,$7,$8,$9,$10,$11,$12,NOW())`,
		newID(), user.ID, periodeID, action, module, description, nullable(entityID), clientIP(r), r.UserAgent(), browserName(r.UserAgent()), deviceName(r.UserAgent()), nullable(location))
	return err
}

func requestLocation(r *http.Request) string {
	cookie, err := r.Cookie("user_address")
	if err != nil {
		return ""
	}
	location, err := url.PathUnescape(cookie.Value)
	if err != nil {
		return cookie.Value
	}
	return location
}

func (a *API) listLogs(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	page, limit := parsePage(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	module := r.URL.Query().Get("module")
	action := r.URL.Query().Get("action")
	userID := r.URL.Query().Get("userId")
	global := r.URL.Query().Get("scope") == "global"
	if global && auth.User.Role != "SEKRETARIS_CABANG" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Akses ditolak")
		return
	}
	var startDate, endDate *time.Time
	if raw := r.URL.Query().Get("startDate"); raw != "" {
		value, err := parseLogDateBoundary(raw, false)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "Rentang tanggal tidak valid")
			return
		}
		startDate = &value
	}
	if raw := r.URL.Query().Get("endDate"); raw != "" {
		value, err := parseLogDateBoundary(raw, true)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "Rentang tanggal tidak valid")
			return
		}
		endDate = &value
	}
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		writeError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "Rentang tanggal tidak valid")
		return
	}
	orderColumn := map[string]string{"action": `l."action"`, "module": `l."module"`, "userName": `u."name"`, "createdAt": `l."createdAt"`}[r.URL.Query().Get("sortKey")]
	if orderColumn == "" {
		orderColumn = `l."createdAt"`
	}
	orderDirection := "DESC"
	if strings.EqualFold(r.URL.Query().Get("sortDir"), "asc") {
		orderDirection = "ASC"
	}
	base := ` FROM "LogActivity" l JOIN "User" u ON u."id"=l."userId" JOIN "Periode" p ON p."id"=l."periodeId"
		WHERE p."isActive"=true
		AND (($1::boolean AND $2::text='SEKRETARIS_CABANG') OR (NOT $1::boolean AND l."userId"=$3))
		AND (NOT $1::boolean OR $4::text='' OR $4='ALL' OR l."userId"=$4)
		AND ($5::text='' OR $5='ALL' OR l."module"::text=$5)
		AND ($6::text='' OR $6='ALL' OR l."action"::text=$6)
		AND ($7::text='' OR l."description" ILIKE '%' || $7 || '%' OR u."name" ILIKE '%' || $7 || '%' OR u."email" ILIKE '%' || $7 || '%' OR COALESCE(l."location",'') ILIKE '%' || $7 || '%')
		AND ($8::timestamptz IS NULL OR l."createdAt">=$8)
		AND ($9::timestamptz IS NULL OR l."createdAt"<=$9)`
	args := []any{global, auth.User.Role, auth.User.ID, userID, module, action, q, startDate, endDate}
	var total int
	if err := a.db.QueryRow(r.Context(), `SELECT COUNT(*)`+base, args...).Scan(&total); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal menghitung log aktivitas")
		return
	}
	if total == 0 || page > totalPages(total, limit) {
		writeJSON(w, 200, map[string]any{"data": []any{}, "total": total, "totalPages": totalPages(total, limit), "currentPage": page})
		return
	}
	query := `SELECT l."id",l."userId",l."periodeId",l."action"::text,l."module"::text,l."description",l."entityId",l."ipAddress",l."userAgent",l."browser",l."device",l."location",l."createdAt",u."name" AS "userName",u."email" AS "userEmail",u."role"::text AS "userRole",p."nama" AS "periodeNama"` + base + ` ORDER BY ` + orderColumn + ` ` + orderDirection + `, l."id" ` + orderDirection + ` LIMIT $10 OFFSET $11`
	rows, err := a.db.Query(r.Context(), query, append(args, limit, (page-1)*limit)...)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil log aktivitas")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil log aktivitas")
		return
	}
	for _, item := range data {
		item["user"] = map[string]any{"name": item["userName"], "email": item["userEmail"], "role": item["userRole"]}
		item["periode"] = map[string]any{"nama": item["periodeNama"]}
	}
	writeJSON(w, 200, map[string]any{"data": data, "total": total, "totalPages": totalPages(total, limit), "currentPage": page})
}

func parseLogDateBoundary(value string, endOfDay bool) (time.Time, error) {
	value = strings.TrimSpace(value)
	if len(value) != len("2006-01-02") {
		return parseDate(value)
	}
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, location)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return parsed.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
	}
	return parsed, nil
}

func (a *API) logStats(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	global := r.URL.Query().Get("scope") == "global"
	if global && auth.User.Role != "SEKRETARIS_CABANG" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Akses ditolak")
		return
	}
	selected := r.URL.Query().Get("userId")
	rows, err := a.db.Query(r.Context(), `SELECT l."module"::text,COUNT(*) FROM "LogActivity" l JOIN "Periode" p ON p."id"=l."periodeId" WHERE p."isActive"=true AND (($1::boolean AND $2::text='SEKRETARIS_CABANG') OR (NOT $1::boolean AND l."userId"=$3)) AND (NOT $1::boolean OR $4::text='' OR $4='ALL' OR l."userId"=$4) GROUP BY l."module"`, global, auth.User.Role, auth.User.ID, selected)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil statistik log")
		return
	}
	counts := map[string]int{"TOTAL": 0, "ARSIP_SURAT": 0, "ANGGOTA": 0, "BERKAS_PIMPINAN": 0, "BERKAS_SP": 0, "AGENDA_KEGIATAN": 0, "PENGAJUAN_BERKAS": 0, "PERIODE": 0, "USER": 0, "AUTH": 0, "PRESENSI": 0}
	for rows.Next() {
		var module string
		var count int
		if err := rows.Scan(&module, &count); err != nil {
			rows.Close()
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca statistik log")
			return
		}
		counts[module] = count
		counts["TOTAL"] += count
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca statistik log")
		return
	}
	writeJSON(w, 200, counts)
}

func (a *API) logMonitoring(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "SEKRETARIS_CABANG") {
		return
	}
	selected := r.URL.Query().Get("userId")
	rows, err := a.db.Query(r.Context(), `SELECT l."module"::text,COUNT(*) FROM "LogActivity" l JOIN "Periode" p ON p."id"=l."periodeId" WHERE p."isActive"=true AND ($1::text='' OR $1='ALL' OR l."userId"=$1) GROUP BY l."module" ORDER BY COUNT(*) DESC`, selected)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil monitoring")
		return
	}
	distribution := []map[string]any{}
	for rows.Next() {
		var module string
		var count int
		if err := rows.Scan(&module, &count); err != nil {
			rows.Close()
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca monitoring")
			return
		}
		distribution = append(distribution, map[string]any{"name": strings.ReplaceAll(module, "_", " "), "value": count})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca monitoring")
		return
	}
	leaderboard := []map[string]any{}
	leaders, err := a.db.Query(r.Context(), `SELECT u."id",u."name",u."image",COUNT(l."id") FROM "LogActivity" l JOIN "Periode" p ON p."id"=l."periodeId" JOIN "User" u ON u."id"=l."userId" WHERE p."isActive"=true AND ($1::text='' OR $1='ALL' OR l."userId"=$1) GROUP BY u."id" ORDER BY COUNT(l."id") DESC LIMIT 5`, selected)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil leaderboard log")
		return
	}
	for leaders.Next() {
		var id, name string
		var image *string
		var count int
		if err := leaders.Scan(&id, &name, &image, &count); err != nil {
			leaders.Close()
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca leaderboard log")
			return
		}
		leaderboard = append(leaderboard, map[string]any{"id": id, "name": name, "image": image, "count": count})
	}
	err = leaders.Err()
	leaders.Close()
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca leaderboard log")
		return
	}
	dayCounts := map[string]int{}
	days, err := a.db.Query(r.Context(), `SELECT to_char(date_trunc('day',l."createdAt" AT TIME ZONE 'Asia/Jakarta'),'YYYY-MM-DD'),COUNT(*) FROM "LogActivity" l JOIN "Periode" p ON p."id"=l."periodeId" WHERE p."isActive"=true AND l."createdAt">=NOW()-INTERVAL '6 days' AND ($1::text='' OR $1='ALL' OR l."userId"=$1) GROUP BY 1`, selected)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil timeline log")
		return
	}
	for days.Next() {
		var date string
		var count int
		if err := days.Scan(&date, &count); err != nil {
			days.Close()
			writeError(w, 500, "DATABASE_ERROR", "Gagal membaca timeline log")
			return
		}
		dayCounts[date] = count
	}
	err = days.Err()
	days.Close()
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca timeline log")
		return
	}
	timeline := make([]map[string]any, 0, 7)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(loc)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		key := day.Format("2006-01-02")
		timeline = append(timeline, map[string]any{"date": day.Format("2 Jan"), "count": dayCounts[key]})
	}
	writeJSON(w, 200, map[string]any{"distribution": distribution, "leaderboard": leaderboard, "timeline": timeline})
}

func (a *API) getLog(w http.ResponseWriter, r *http.Request) {
	auth := authFrom(r)
	id := chi.URLParam(r, "id")
	rows, err := a.db.Query(r.Context(), `SELECT l."id",l."userId",l."periodeId",l."action"::text,l."module"::text,l."description",l."entityId",l."ipAddress",l."userAgent",l."browser",l."device",l."location",l."createdAt",u."name" AS "userName",u."email" AS "userEmail",u."role"::text AS "userRole",p."nama" AS "periodeNama" FROM "LogActivity" l JOIN "User" u ON u."id"=l."userId" JOIN "Periode" p ON p."id"=l."periodeId" WHERE l."id"=$1 AND ($2::text='SEKRETARIS_CABANG' OR l."userId"=$3)`, id, auth.User.Role, auth.User.ID)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mengambil log")
		return
	}
	data, err := scanMaps(rows)
	if err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal membaca log")
		return
	}
	if len(data) == 0 {
		writeError(w, 404, "NOT_FOUND", "Log tidak ditemukan")
		return
	}
	item := data[0]
	item["user"] = map[string]any{"name": item["userName"], "email": item["userEmail"], "role": item["userRole"]}
	item["periode"] = map[string]any{"nama": item["periodeNama"]}
	writeJSON(w, 200, map[string]any{"data": item})
}

func (a *API) createClientLog(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action      string `json:"action"`
		Module      string `json:"module"`
		Description string `json:"description"`
		EntityID    string `json:"entityId"`
	}
	if readJSON(r, &body) != nil || body.Action != "EXPORT" || !oneOf(body.Module, "ARSIP_SURAT", "ANGGOTA", "BERKAS_PIMPINAN", "BERKAS_SP", "AGENDA_KEGIATAN", "PENGAJUAN_BERKAS", "PERIODE", "USER", "AUTH", "PRESENSI") {
		writeError(w, 400, "INVALID_REQUEST", "Log tidak valid")
		return
	}
	body.Description = strings.TrimSpace(body.Description)
	if len(body.Description) > 2000 || len(body.EntityID) > 256 {
		writeError(w, 400, "INVALID_REQUEST", "Log terlalu panjang")
		return
	}
	auth := authFrom(r)
	if err := a.createLog(r.Context(), r, auth.User, body.Action, body.Module, body.Description, body.EntityID); err != nil {
		writeError(w, 500, "DATABASE_ERROR", "Gagal mencatat aktivitas")
		return
	}
	writeJSON(w, 200, map[string]any{"success": true})
}

func browserName(ua string) string {
	lower := strings.ToLower(ua)
	switch {
	case strings.Contains(lower, "edg/"):
		return "Edge"
	case strings.Contains(lower, "chrome/"):
		return "Chrome"
	case strings.Contains(lower, "firefox/"):
		return "Firefox"
	case strings.Contains(lower, "safari/"):
		return "Safari"
	default:
		return "Unknown"
	}
}
func deviceName(ua string) string {
	lower := strings.ToLower(ua)
	if strings.Contains(lower, "mobile") || strings.Contains(lower, "android") || strings.Contains(lower, "iphone") {
		return "Mobile"
	}
	return "Desktop"
}
