package main

import (
	"database/sql"
	"fmt"
	"net/http"
)

// Único propósito: provar que o que o webhook indexou é lível de volta.
// Sem estilo, sem paginação — é o critério mínimo de "Beta 0 concluído"
// (docs/BETA0-PLAN.md), não uma UI de produto.
func setupList(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("GET /shelf", func(w http.ResponseWriter, r *http.Request) {
		items, err := listShelfItems(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<!doctype html><h1>Estante sincronizada</h1><ul>")
		for _, it := range items {
			fmt.Fprintf(w, "<li><b>%s/%s</b> — %s (indexado %s)<br><small>%s</small></li>",
				it.Provider, it.WorkID, it.DID, it.IndexedAt, it.URI)
		}
		if len(items) == 0 {
			fmt.Fprint(w, "<li>nada sincronizado ainda</li>")
		}
		fmt.Fprint(w, "</ul>")
	})
}
