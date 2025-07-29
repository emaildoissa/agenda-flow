package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/emaildoissa/agenda-flow/internal/models"
	"github.com/go-chi/chi/v5"
)

// AgendamentosHandler gerencia as requisições relacionadas a agendamentos.
type AgendamentosHandler struct {
	DB *sql.DB
}

// NewAgendamentosHandler cria uma nova instância de AgendamentosHandler.
func NewAgendamentosHandler(db *sql.DB) *AgendamentosHandler {
	return &AgendamentosHandler{DB: db}
}

// CreateAgendamento cria uma nova solicitação de agendamento.
func (h *AgendamentosHandler) CreateAgendamento(w http.ResponseWriter, r *http.Request) {
	// 1. Decodificar o corpo da requisição.
	var agendamento models.Agendamento
	err := json.NewDecoder(r.Body).Decode(&agendamento)
	if err != nil {
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	// 2. Buscar a duração do serviço para calcular a hora de término.
	var duracaoMinutos int
	err = h.DB.QueryRow("SELECT duracao_minutos FROM servicos WHERE id = $1", agendamento.ServicoID).Scan(&duracaoMinutos)
	if err != nil {
		log.Printf("Erro ao buscar duração do serviço: %v", err)
		http.Error(w, "Serviço inválido", http.StatusBadRequest)
		return
	}

	// 3. Calcular a data_hora_fim e definir o status inicial.
	agendamento.DataHoraFim = agendamento.DataHoraInicio.Add(time.Duration(duracaoMinutos) * time.Minute)
	agendamento.Status = "PENDENTE"

	// 4. Inserir no banco de dados.
	sqlStatement := `
		INSERT INTO agendamentos (salao_id, servico_id, cliente_nome, cliente_contato, data_hora_inicio, data_hora_fim, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, criado_em`

	err = h.DB.QueryRow(
		sqlStatement,
		agendamento.SalaoID, agendamento.ServicoID, agendamento.ClienteNome, agendamento.ClienteContato,
		agendamento.DataHoraInicio, agendamento.DataHoraFim, agendamento.Status,
	).Scan(&agendamento.ID, &agendamento.CriadoEm)

	if err != nil {
		log.Printf("Erro ao inserir agendamento: %v", err)
		http.Error(w, "Erro ao criar agendamento", http.StatusInternalServerError)
		return
	}

	// 5. SIMULAR O GATILHO PARA O N8N
	// Em um cenário real, aqui faríamos uma chamada HTTP para o webhook do n8n.
	// Por enquanto, vamos apenas logar a informação.
	log.Printf("!!! GATILHO N8N: Novo agendamento pendente ID %d para o salão %d. Notificar dono!", agendamento.ID, agendamento.SalaoID)

	// 6. Responder com o agendamento criado.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agendamento)
}

// updateAgendamentoStatus é uma função auxiliar para não repetir código.
func (h *AgendamentosHandler) updateAgendamentoStatus(w http.ResponseWriter, r *http.Request, novoStatus string) {
	// 1. Pegar o ID do agendamento da URL.
	agendamentoIDStr := chi.URLParam(r, "idAgendamento")
	agendamentoID, err := strconv.Atoi(agendamentoIDStr)
	if err != nil {
		http.Error(w, "ID de agendamento inválido", http.StatusBadRequest)
		return
	}

	// 2. Atualizar o status no banco.
	sqlStatement := `UPDATE agendamentos SET status = $1 WHERE id = $2`
	res, err := h.DB.Exec(sqlStatement, novoStatus, agendamentoID)
	if err != nil {
		log.Printf("Erro ao atualizar status do agendamento: %v", err)
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	// Verificar se alguma linha foi realmente atualizada.
	count, err := res.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}
	if count == 0 {
		http.Error(w, "Agendamento não encontrado", http.StatusNotFound)
		return
	}

	// 3. SIMULAR A NOTIFICAÇÃO DE VOLTA PARA O CLIENTE
	log.Printf("!!! GATILHO N8N: Agendamento ID %d foi atualizado para %s. Notificar cliente final!", agendamentoID, novoStatus)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sucesso", "novo_status": novoStatus})
}

// ConfirmarAgendamento atualiza o status para 'CONFIRMADO'.
func (h *AgendamentosHandler) ConfirmarAgendamento(w http.ResponseWriter, r *http.Request) {
	h.updateAgendamentoStatus(w, r, "CONFIRMADO")
}

// CancelarAgendamento atualiza o status para 'CANCELADO'.
func (h *AgendamentosHandler) CancelarAgendamento(w http.ResponseWriter, r *http.Request) {
	h.updateAgendamentoStatus(w, r, "CANCELADO")
}
