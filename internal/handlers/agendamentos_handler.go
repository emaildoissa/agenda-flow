package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/emaildoissa/agenda-flow/internal/models"
	"github.com/go-chi/chi/v5"
)

// AgendamentosHandler agora segura a URL do webhook do n8n
type AgendamentosHandler struct {
	DB            *sql.DB
	N8NWebhookURL string
}

// NewAgendamentosHandler é o construtor para nosso handler
func NewAgendamentosHandler(db *sql.DB, n8nWebhookURL string) *AgendamentosHandler {
	return &AgendamentosHandler{
		DB:            db,
		N8NWebhookURL: n8nWebhookURL,
	}
}

// N8NPayload é a estrutura de dados que enviaremos para o n8n
type N8NPayload struct {
	AgendamentoID       int    `json:"agendamento_id"`
	ClienteNome         string `json:"cliente_nome"`
	ServicoNome         string `json:"servico_nome"`
	DataHoraFormatada   string `json:"data_hora_formatada"`
	WhatsappNotificacao string `json:"whatsapp_notificacao"`
}

// CreateAgendamento cria um agendamento e dispara o gatilho para o n8n
func (h *AgendamentosHandler) CreateAgendamento(w http.ResponseWriter, r *http.Request) {
	var agendamento models.Agendamento
	err := json.NewDecoder(r.Body).Decode(&agendamento)
	if err != nil {
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	var duracaoMinutos int
	var nomeServico string
	err = h.DB.QueryRow("SELECT nome, duracao_minutos FROM servicos WHERE id = $1 AND salao_id = $2", agendamento.ServicoID, agendamento.SalaoID).Scan(&nomeServico, &duracaoMinutos)
	if err != nil {
		http.Error(w, "Serviço inválido", http.StatusBadRequest)
		return
	}

	var whatsappNotificacao string
	err = h.DB.QueryRow("SELECT whatsapp_notificacao FROM saloes WHERE id = $1", agendamento.SalaoID).Scan(&whatsappNotificacao)
	if err != nil {
		http.Error(w, "Salão inválido", http.StatusBadRequest)
		return
	}

	agendamento.DataHoraFim = agendamento.DataHoraInicio.Add(time.Duration(duracaoMinutos) * time.Minute)
	agendamento.Status = "CONFIRMADO"

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

	// Prepara os dados para o n8n
	location, _ := time.LoadLocation("America/Sao_Paulo")
	payload := N8NPayload{
		AgendamentoID:       agendamento.ID,
		ClienteNome:         agendamento.ClienteNome,
		ServicoNome:         nomeServico,
		DataHoraFormatada:   agendamento.DataHoraInicio.In(location).Format("15:04 de 02/01/2006"),
		WhatsappNotificacao: whatsappNotificacao,
	}

	// Dispara o webhook em uma goroutine para não bloquear a resposta ao usuário
	go h.dispararWebhookN8N(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agendamento)
}

// dispararWebhookN8N envia os dados do agendamento para a URL do webhook do n8n.
func (h *AgendamentosHandler) dispararWebhookN8N(payload N8NPayload) {
	if h.N8NWebhookURL == "" {
		log.Println("AVISO: URL do webhook do n8n não configurada. Pulando notificação.")
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Erro ao serializar payload para o n8n: %v", err)
		return
	}

	req, err := http.NewRequest("POST", h.N8NWebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Erro ao criar requisição para o n8n: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Erro ao enviar webhook para o n8n: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Erro do n8n ao receber webhook. Status: %s", resp.Status)
		return
	}

	log.Printf("Webhook enviado para o n8n com sucesso. Status: %s", resp.Status)
}

// As funções abaixo não foram modificadas
func (h *AgendamentosHandler) updateAgendamentoStatus(w http.ResponseWriter, r *http.Request, novoStatus string) {
	agendamentoIDStr := chi.URLParam(r, "idAgendamento")
	agendamentoID, err := strconv.Atoi(agendamentoIDStr)
	if err != nil {
		http.Error(w, "ID de agendamento inválido", http.StatusBadRequest)
		return
	}

	sqlStatement := `UPDATE agendamentos SET status = $1 WHERE id = $2`
	res, err := h.DB.Exec(sqlStatement, novoStatus, agendamentoID)
	if err != nil {
		log.Printf("Erro ao atualizar status do agendamento: %v", err)
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

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

	log.Printf("!!! GATILHO N8N: Agendamento ID %d foi atualizado para %s. Notificar cliente final!", agendamentoID, novoStatus)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sucesso", "novo_status": novoStatus})
}

func (h *AgendamentosHandler) ConfirmarAgendamento(w http.ResponseWriter, r *http.Request) {
	h.updateAgendamentoStatus(w, r, "CONFIRMADO")
}

func (h *AgendamentosHandler) CancelarAgendamento(w http.ResponseWriter, r *http.Request) {
	h.updateAgendamentoStatus(w, r, "CANCELADO")
}
