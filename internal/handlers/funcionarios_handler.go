// internal/handlers/funcionarios_handler.go

package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/emaildoissa/agenda-flow/internal/models"
	"github.com/go-chi/chi/v5"
)

type FuncionariosHandler struct {
	DB *sql.DB
}

func NewFuncionariosHandler(db *sql.DB) *FuncionariosHandler {
	return &FuncionariosHandler{DB: db}
}

// ListFuncionariosBySalaoID lista todos os funcionários ativos de um salão.
func (h *FuncionariosHandler) ListFuncionariosBySalaoID(w http.ResponseWriter, r *http.Request) {
	salaoIDStr := chi.URLParam(r, "idSalao")
	salaoID, err := strconv.Atoi(salaoIDStr)
	if err != nil {
		http.Error(w, "ID de salão inválido", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query("SELECT id, nome, ativo FROM funcionarios WHERE salao_id = $1 AND ativo = TRUE", salaoID)
	if err != nil {
		log.Printf("Erro ao buscar funcionários: %v", err)
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var funcionarios []models.Funcionario
	for rows.Next() {
		var f models.Funcionario
		if err := rows.Scan(&f.ID, &f.Nome, &f.Ativo); err != nil {
			log.Printf("Erro ao escanear funcionário: %v", err)
			http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
			return
		}
		funcionarios = append(funcionarios, f)
	}

	if funcionarios == nil {
		funcionarios = make([]models.Funcionario, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(funcionarios)
}
