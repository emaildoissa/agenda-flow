package models

import (
	"time"
)

type Salao struct {
	ID                    int       `json:"id"`
	NomeSalao             string    `json:"nome_salao"`
	EmailProprietario     string    `json:"email_proprietario"`
	HashSenha             string    `json:"senha"`
	WhatsappNotificacao   string    `json:"whatsapp_notificacao"`
	HorariosFuncionamento []byte    `json:"horarios_funcionamento"` // Representado como JSON raw
	CriadoEm              time.Time `json:"criado_em"`
}

type Servico struct {
	ID             int     `json:"id"`
	SalaoID        int     `json:"salao_id"`
	Nome           string  `json:"nome"`
	DuracaoMinutos int     `json:"duracao_minutos"`
	Preco          float64 `json:"preco"`
	Ativo          bool    `json:"ativo"`
}

type Agendamento struct {
	ID             int       `json:"id"`
	SalaoID        int       `json:"salao_id"`
	ServicoID      int       `json:"servico_id"`
	ClienteNome    string    `json:"cliente_nome"`
	ClienteContato string    `json:"cliente_contato"`
	DataHoraInicio time.Time `json:"data_hora_inicio"`
	DataHoraFim    time.Time `json:"data_hora_fim"`
	Status         string    `json:"status"`
	CriadoEm       time.Time `json:"criado_em"`
}
type Funcionario struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Ativo bool   `json:"ativo"`
}
