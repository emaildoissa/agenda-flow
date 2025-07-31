package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emaildoissa/agenda-flow/internal/models"
	"github.com/go-chi/chi/v5"
)

// Estruturas para decodificar o JSON de horários
type HorarioDia struct {
	Inicio string  `json:"inicio"`
	Fim    string  `json:"fim"`
	Pausas []Pausa `json:"pausas"`
}
type Pausa struct {
	Inicio string `json:"inicio"`
	Fim    string `json:"fim"`
}

type DisponibilidadeHandler struct {
	DB *sql.DB
}

func NewDisponibilidadeHandler(db *sql.DB) *DisponibilidadeHandler {
	return &DisponibilidadeHandler{DB: db}
}

func (h *DisponibilidadeHandler) GetDisponibilidade(w http.ResponseWriter, r *http.Request) {
	salaoIDStr := chi.URLParam(r, "idSalao")
	salaoID, _ := strconv.Atoi(salaoIDStr)
	dataStr := r.URL.Query().Get("data")
	servicoIDStr := r.URL.Query().Get("servicoId")
	servicoID, _ := strconv.Atoi(servicoIDStr)

	data, err := time.ParseInLocation("2006-01-02", dataStr, time.UTC)
	if err != nil {
		http.Error(w, "Formato de data inválido. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	var duracaoMinutos int
	err = h.DB.QueryRow("SELECT duracao_minutos FROM servicos WHERE id = $1", servicoID).Scan(&duracaoMinutos)
	if err != nil {
		http.Error(w, "Serviço não encontrado", http.StatusNotFound)
		return
	}

	var horariosJSON []byte
	err = h.DB.QueryRow("SELECT horarios_funcionamento FROM saloes WHERE id = $1", salaoID).Scan(&horariosJSON)
	if err != nil || horariosJSON == nil {
		http.Error(w, "Horários de funcionamento não configurados", http.StatusNotFound)
		return
	}

	var todosHorarios map[string]*HorarioDia
	json.Unmarshal(horariosJSON, &todosHorarios)
	agendamentos, err := h.getAgendamentosDoDia(salaoID, data)
	if err != nil {
		http.Error(w, "Erro ao buscar agendamentos", http.StatusInternalServerError)
		return
	}
	log.Printf("Para a data %s, foram encontrados %d agendamentos.", dataStr, len(agendamentos))

	diaDaSemanaIngles := strings.ToLower(data.Weekday().String())
	mapaDias := map[string]string{
		"sunday": "domingo", "monday": "segunda", "tuesday": "terca",
		"wednesday": "quarta", "thursday": "quinta", "friday": "sexta",
		"saturday": "sabado",
	}
	chaveDiaPt := mapaDias[diaDaSemanaIngles]
	horarioDoDia := todosHorarios[chaveDiaPt]

	if horarioDoDia == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
		return
	}

	slotsDisponiveis := h.calcularSlots(data, horarioDoDia, duracaoMinutos, agendamentos)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slotsDisponiveis)
}

func (h *DisponibilidadeHandler) getAgendamentosDoDia(salaoID int, data time.Time) ([]models.Agendamento, error) {
	inicioDia := time.Date(data.Year(), data.Month(), data.Day(), 0, 0, 0, 0, time.UTC)
	fimDia := inicioDia.Add(24 * time.Hour)

	rows, err := h.DB.Query(`
		SELECT id, data_hora_inicio, data_hora_fim, status FROM agendamentos
		WHERE salao_id = $1 AND data_hora_inicio >= $2 AND data_hora_inicio < $3 AND status IN ('CONFIRMADO', 'PENDENTE')`,
		salaoID, inicioDia, fimDia)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agendamentos []models.Agendamento
	for rows.Next() {
		var a models.Agendamento
		if err := rows.Scan(&a.ID, &a.DataHoraInicio, &a.DataHoraFim, &a.Status); err != nil {
			return nil, err
		}
		agendamentos = append(agendamentos, a)
	}
	return agendamentos, nil
}

func (h *DisponibilidadeHandler) calcularSlots(dataDoAgendamento time.Time, horarioUtil *HorarioDia, duracaoServico int, agendamentosExistentes []models.Agendamento) []string {
	var slotsDisponiveis []string
	loc := time.UTC

	parseTime := func(t string) time.Time {
		parsed, _ := time.Parse("15:04", t)
		return time.Date(dataDoAgendamento.Year(), dataDoAgendamento.Month(), dataDoAgendamento.Day(), parsed.Hour(), parsed.Minute(), 0, 0, loc)
	}

	inicioDia := parseTime(horarioUtil.Inicio)
	fimDia := parseTime(horarioUtil.Fim)

	slotAtual := inicioDia
	for slotAtual.Before(fimDia) {
		fimSlot := slotAtual.Add(time.Duration(duracaoServico) * time.Minute)
		if fimSlot.After(fimDia) {
			break
		}

		temConflito := false

		// VERIFICA CONFLITO COM AGENDAMENTOS EXISTENTES
		for _, agendamento := range agendamentosExistentes {
			// <<< LOG DE DEPURAÇÃO ADICIONADO AQUI >>>
			log.Printf("--- Checando Slot: %s ---", slotAtual.Format("15:04"))
			log.Printf("SLOT INÍCIO: %s", slotAtual.Format(time.RFC3339))
			log.Printf("SLOT FIM:    %s", fimSlot.Format(time.RFC3339))
			log.Printf("AGENDADO INÍCIO: %s", agendamento.DataHoraInicio.Format(time.RFC3339))
			log.Printf("AGENDADO FIM:    %s", agendamento.DataHoraFim.Format(time.RFC3339))

			if slotAtual.Before(agendamento.DataHoraFim) && fimSlot.After(agendamento.DataHoraInicio) {
				log.Printf("--> CONFLITO DETECTADO!")
				temConflito = true
				break
			}
		}

		// VERIFICA CONFLITO COM PAUSAS (lógica omitida para simplificar, mas a estrutura é a mesma)

		if !temConflito {
			slotsDisponiveis = append(slotsDisponiveis, slotAtual.Format("15:04"))
		}

		slotAtual = slotAtual.Add(15 * time.Minute)
	}

	return slotsDisponiveis
}
