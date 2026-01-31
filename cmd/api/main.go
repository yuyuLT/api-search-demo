package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Meta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Count   int `json:"count"`
}

type Response struct {
	Meta  Meta              `json:"meta"`
	Items []map[string]any  `json:"items"`
}

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is required (see .env.example)")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			log.Println("db ping error:", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/v1/items", func(w http.ResponseWriter, r *http.Request) {
		handleItems(w, r, db)
	})

	srv := &http.Server{
		Addr:              "127.0.0.1:" + port,
		Handler:           logMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://%s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func handleItems(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	q := r.URL.Query()
	category := strings.TrimSpace(q.Get("category"))
	material := strings.TrimSpace(q.Get("material"))

	perPage := parseIntDefault(q.Get("per_page"), 50)
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}

	afterID := parseInt64Default(q.Get("after_id"), 0) // 0は未指定扱い

	where := []string{"1=1"}
	args := []any{}

	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if material != "" {
		where = append(where, "material = ?")
		args = append(args, material)
	}

	// seek pagination
	if afterID > 0 {
		where = append(where, "id < ?")
		args = append(args, afterID)
	}

	sqlStr := fmt.Sprintf(`
SELECT * FROM items
WHERE %s
ORDER BY id DESC
LIMIT ?`, strings.Join(where, " AND "))

	args = append(args, perPage)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		log.Println("db query error:", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "db query failed"})
		return
	}
	defer rows.Close()

	items, err := rowsToMaps(rows)
	if err != nil {
		log.Println("db scan error:", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "db scan failed"})
		return
	}

	var nextAfterID any = nil
	if len(items) > 0 {
		// itemsは id DESC なので、末尾が次の after_id になる
		if v, ok := items[len(items)-1]["id"]; ok {
			nextAfterID = v
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"meta": map[string]any{
			"per_page":       perPage,
			"count":          len(items),
			"next_after_id":  nextAfterID,
		},
		"items": items,
	})
}

func parseInt64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}


func rowsToMaps(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := make(map[string]any, len(cols))
		for i, c := range cols {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				row[c] = string(b)
			} else {
				row[c] = v
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
