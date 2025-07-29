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

// ServicosHandler gerencia as requisições relacionadas a serviços.
type ServicosHandler struct {
	DB *sql.DB
}

// NewServicosHandler cria uma nova instância de ServicosHandler.
func NewServicosHandler(db *sql.DB) *ServicosHandler {
	return &ServicosHandler{DB: db}
}

// CreateServico adiciona um novo serviço a um salão.
func (h *ServicosHandler) CreateServico(w http.ResponseWriter, r *http.Request) {
	// 1. Pegar o ID do salão da URL.
	salaoIDStr := chi.URLParam(r, "idSalao")
	salaoID, err := strconv.Atoi(salaoIDStr)
	if err != nil {
		http.Error(w, "ID de salão inválido", http.StatusBadRequest)
		return
	}

	// 2. Decodificar o corpo da requisição.
	var servico models.Servico
	err = json.NewDecoder(r.Body).Decode(&servico)
	if err != nil {
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	// 3. Inserir no banco de dados.
	servico.SalaoID = salaoID // Garante que o serviço seja associado ao salão correto.
	sqlStatement := `
		INSERT INTO servicos (salao_id, nome, duracao_minutos, preco)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err = h.DB.QueryRow(sqlStatement, servico.SalaoID, servico.Nome, servico.DuracaoMinutos, servico.Preco).Scan(&servico.ID)
	if err != nil {
		log.Printf("Erro ao inserir serviço: %v", err)
		http.Error(w, "Erro ao criar serviço", http.StatusInternalServerError)
		return
	}

	// 4. Responder com o serviço criado.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(servico)
}

// ListServicosBySalaoID lista todos os serviços de um salão específico.
func (h *ServicosHandler) ListServicosBySalaoID(w http.ResponseWriter, r *http.Request) {
	// 1. Pegar o ID do salão da URL.
	salaoIDStr := chi.URLParam(r, "idSalao")
	salaoID, err := strconv.Atoi(salaoIDStr)
	if err != nil {
		http.Error(w, "ID de salão inválido", http.StatusBadRequest)
		return
	}

	// 2. Buscar os serviços no banco.
	rows, err := h.DB.Query("SELECT id, salao_id, nome, duracao_minutos, preco, ativo FROM servicos WHERE salao_id = $1 AND ativo = TRUE", salaoID)
	if err != nil {
		log.Printf("Erro ao buscar serviços: %v", err)
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// 3. Iterar sobre os resultados e construir a lista.
	var servicos []models.Servico
	for rows.Next() {
		var s models.Servico
		if err := rows.Scan(&s.ID, &s.SalaoID, &s.Nome, &s.DuracaoMinutos, &s.Preco, &s.Ativo); err != nil {
			log.Printf("Erro ao escanear serviço: %v", err)
			http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
			return
		}
		servicos = append(servicos, s)
	}

	// Se 'servicos' for nulo (nenhum encontrado), o Go o codificará como 'null' em JSON.
	// Para garantir que sempre retornemos uma lista vazia `[]`, fazemos esta verificação.
	if servicos == nil {
		servicos = make([]models.Servico, 0)
	}

	// 4. Responder com a lista.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(servicos)
}
