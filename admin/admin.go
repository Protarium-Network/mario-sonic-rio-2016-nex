// Package rio2016admin serves a small password-protected web page for
// manually managing Mario & Sonic Rio 2016 leaderboard data (add/edit/delete
// ranking rows) without needing direct SQL access. Intentionally minimal:
// server-rendered HTML, no JS framework, no client library.
package admin

import (
	"encoding/hex"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/database"
	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/globals"
)

var adminPassword string

// Start launches the admin HTTP server. Blocks; call in its own goroutine.
func Start() {
	adminPassword = strings.TrimSpace(os.Getenv("PN_RIO2016_ADMIN_PASSWORD"))
	if adminPassword == "" {
		globals.Logger.Warning("[Rio2016 Admin] PN_RIO2016_ADMIN_PASSWORD not set - admin panel disabled")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", requireAuth(handleIndex))
	mux.HandleFunc("/add", requireAuth(handleAdd))
	mux.HandleFunc("/delete", requireAuth(handleDelete))

	listenAddress := os.Getenv("PN_RIO2016_ADMIN_LISTEN")
	if listenAddress == "" {
		listenAddress = ":8090"
	}

	globals.Logger.Successf("[Rio2016 Admin] Listening on %s", listenAddress)
	if err := http.ListenAndServe(listenAddress, mux); err != nil {
		globals.Logger.Error(err.Error())
	}
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, password, ok := r.BasicAuth()
		if !ok || password != adminPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="rio2016-admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

type rankingRow struct {
	Category    int64
	OwnerPID    int64
	UniqueID    int64
	Score       int64
	OrderBy     int64
	Groups      string
	Param       int64
	HasCommon   bool
	OrderByText string
}

var pageTemplate = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="utf-8">
<title>Rio 2016 - Admin classements</title>
<style>
	body { font-family: -apple-system, sans-serif; max-width: 900px; margin: 2rem auto; padding: 0 1rem; background: #14161a; color: #e6e6e6; }
	h1 { font-size: 1.3rem; }
	table { border-collapse: collapse; width: 100%; margin-bottom: 2rem; }
	th, td { border: 1px solid #333; padding: 0.4rem 0.6rem; text-align: left; font-size: 0.85rem; }
	th { background: #1f2228; }
	tr:nth-child(even) { background: #191b20; }
	form.inline { display: inline; }
	button { cursor: pointer; background: #d33; color: white; border: none; padding: 0.2rem 0.5rem; border-radius: 3px; }
	.addform { background: #1f2228; padding: 1rem; border-radius: 6px; margin-bottom: 2rem; }
	.addform label { display: block; margin-top: 0.5rem; font-size: 0.8rem; color: #aaa; }
	.addform input, .addform select { width: 100%; padding: 0.3rem; box-sizing: border-box; background: #14161a; color: #eee; border: 1px solid #444; border-radius: 3px; }
	.addform .row { display: flex; gap: 1rem; }
	.addform .row > div { flex: 1; }
	.submit { margin-top: 1rem; background: #2a7; padding: 0.5rem 1rem; border-radius: 4px; }
	h2 { font-size: 1rem; margin-top: 2rem; border-bottom: 1px solid #333; padding-bottom: 0.3rem; }
	.hint { color: #888; font-size: 0.75rem; }
</style>
</head>
<body>
<h1>🏅 Mario &amp; Sonic Rio 2016 — Classements</h1>

<div class="addform">
<h2 style="margin-top:0;border:none;">Ajouter / modifier un score</h2>
<form method="POST" action="/add">
	<div class="row">
		<div><label>Catégorie (sport)</label><input name="category" type="number" required></div>
		<div><label>PID joueur</label><input name="owner_pid" type="number" required></div>
		<div><label>Unique ID</label><input name="unique_id" type="number" value="1" required></div>
	</div>
	<div class="row">
		<div><label>Score</label><input name="score" type="number" required></div>
		<div><label>Ordre</label>
			<select name="order_by">
				<option value="0">0 - Croissant (temps, plus petit = mieux)</option>
				<option value="1">1 - Décroissant (points, plus grand = mieux)</option>
			</select>
		</div>
		<div><label>Param (data_id ghost, 0 = aucun)</label><input name="param" type="number" value="0"></div>
	</div>
	<div class="hint">Le champ Groups est fixé automatiquement à 39000000 (format observé sur les vraies soumissions).</div>
	<button class="submit" type="submit">Enregistrer</button>
</form>
</div>

{{range .Categories}}
<h2>Catégorie {{.Category}}</h2>
<table>
<tr><th>PID</th><th>UniqueID</th><th>Score</th><th>Ordre</th><th>Param</th><th>Common data</th><th></th></tr>
{{range .Rows}}
<tr>
	<td>{{.OwnerPID}}</td>
	<td>{{.UniqueID}}</td>
	<td>{{.Score}}</td>
	<td>{{.OrderByText}}</td>
	<td>{{.Param}}</td>
	<td>{{if .HasCommon}}oui{{else}}—{{end}}</td>
	<td>
		<form class="inline" method="POST" action="/delete" onsubmit="return confirm('Supprimer ce score ?');">
			<input type="hidden" name="category" value="{{.Category}}">
			<input type="hidden" name="owner_pid" value="{{.OwnerPID}}">
			<input type="hidden" name="unique_id" value="{{.UniqueID}}">
			<button type="submit">Suppr</button>
		</form>
	</td>
</tr>
{{end}}
</table>
{{end}}

</body>
</html>`))

type categoryGroup struct {
	Category int64
	Rows     []rankingRow
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Postgres.Query(`
		SELECT r.category, r.owner_pid, r.unique_id, r.score, r.order_by, r.param,
			EXISTS(SELECT 1 FROM rio2016_common_datas cd WHERE cd.owner_pid = r.owner_pid AND cd.unique_id = r.unique_id) AS has_common
		FROM rio2016_rankings r
		ORDER BY r.category, r.order_by, r.score
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	byCategory := map[int64]*categoryGroup{}
	var order []int64

	for rows.Next() {
		var rr rankingRow
		if err := rows.Scan(&rr.Category, &rr.OwnerPID, &rr.UniqueID, &rr.Score, &rr.OrderBy, &rr.Param, &rr.HasCommon); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if rr.OrderBy == 1 {
			rr.OrderByText = "1 (desc)"
		} else {
			rr.OrderByText = "0 (asc)"
		}
		grp, ok := byCategory[rr.Category]
		if !ok {
			grp = &categoryGroup{Category: rr.Category}
			byCategory[rr.Category] = grp
			order = append(order, rr.Category)
		}
		grp.Rows = append(grp.Rows, rr)
	}

	data := struct{ Categories []*categoryGroup }{}
	for _, cat := range order {
		data.Categories = append(data.Categories, byCategory[cat])
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplate.Execute(w, data); err != nil {
		globals.Logger.Error(err.Error())
	}
}

func handleAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category, err1 := strconv.ParseInt(r.FormValue("category"), 10, 64)
	ownerPID, err2 := strconv.ParseInt(r.FormValue("owner_pid"), 10, 64)
	uniqueID, err3 := strconv.ParseInt(r.FormValue("unique_id"), 10, 64)
	score, err4 := strconv.ParseInt(r.FormValue("score"), 10, 64)
	orderBy, err5 := strconv.ParseInt(r.FormValue("order_by"), 10, 64)
	param, err6 := strconv.ParseInt(r.FormValue("param"), 10, 64)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
		http.Error(w, "invalid form values", http.StatusBadRequest)
		return
	}

	// Fixed groups value matching the format observed on every real client
	// submission (39 00 00 00 / 39 0a 00 00) - keeps admin-added rows
	// indistinguishable from real ones on the wire.
	groups, _ := hex.DecodeString("39000000")

	now := time.Now().Unix()

	tx, err := database.Postgres.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO rio2016_ranking_categories (category, order_by, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (category) DO NOTHING
	`, category, orderBy, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO rio2016_rankings (owner_pid, unique_id, category, score, order_by, update_mode, groups, param, updated_at)
		VALUES ($1, $2, $3, $4, $5, 0, $6, $7, $8)
		ON CONFLICT (unique_id, owner_pid, category) DO UPDATE SET
			score = EXCLUDED.score,
			order_by = EXCLUDED.order_by,
			param = EXCLUDED.param,
			updated_at = EXCLUDED.updated_at
	`, ownerPID, uniqueID, category, score, orderBy, groups, param, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	globals.Logger.Infof("[Rio2016 Admin] added/updated ranking: category=%d owner_pid=%d unique_id=%d score=%d", category, ownerPID, uniqueID, score)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category, err1 := strconv.ParseInt(r.FormValue("category"), 10, 64)
	ownerPID, err2 := strconv.ParseInt(r.FormValue("owner_pid"), 10, 64)
	uniqueID, err3 := strconv.ParseInt(r.FormValue("unique_id"), 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		http.Error(w, "invalid form values", http.StatusBadRequest)
		return
	}

	_, err := database.Postgres.Exec(`
		DELETE FROM rio2016_rankings WHERE category = $1 AND owner_pid = $2 AND unique_id = $3
	`, category, ownerPID, uniqueID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	globals.Logger.Infof("[Rio2016 Admin] deleted ranking: category=%d owner_pid=%d unique_id=%d", category, ownerPID, uniqueID)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
