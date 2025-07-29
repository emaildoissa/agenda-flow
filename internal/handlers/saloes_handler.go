package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/emaildoissa/agenda-flow/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// SaloesHandler é uma struct que segura as dependências, como o banco de dados.
// Isso facilita os testes e a organização.
type SaloesHandler struct {
	DB *sql.DB
}

// NewSaloesHandler cria uma nova instância de SaloesHandler.
func NewSaloesHandler(db *sql.DB) *SaloesHandler {
	return &SaloesHandler{DB: db}
}

// CreateSalao é o método que gerencia a requisição para criar um novo salão.
func (h *SaloesHandler) CreateSalao(w http.ResponseWriter, r *http.Request) {
	// 1. Decodificar o JSON do corpo da requisição para o nosso struct models.Salao
	var salao models.Salao
	err := json.NewDecoder(r.Body).Decode(&salao)
	if err != nil {
		http.Error(w, "Corpo da requisição inválido", http.StatusBadRequest)
		return
	}

	// 2. Validar os dados recebidos (exemplo simples)
	if salao.EmailProprietario == "" || salao.NomeSalao == "" || salao.WhatsappNotificacao == "" {
		http.Error(w, "Campos obrigatórios estão faltando", http.StatusBadRequest)
		return
	}
	// A senha será recebida no campo HashSenha, mas em texto plano por enquanto
	if salao.HashSenha == "" {
		http.Error(w, "Senha é obrigatória", http.StatusBadRequest)
		return
	}

	// 3. Hashear a senha recebida usando bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(salao.HashSenha), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Erro ao hashear a senha: %v", err)
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}
	salao.HashSenha = string(hashedPassword) // Armazena o hash

	// 4. Inserir o novo salão no banco de dados
	sqlStatement := `
		INSERT INTO saloes (nome_salao, email_proprietario, hash_senha, whatsapp_notificacao, horarios_funcionamento)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, criado_em`

	// Usamos QueryRow para pegar o ID e criado_em que foram gerados pelo banco
	err = h.DB.QueryRow(
		sqlStatement,
		salao.NomeSalao,
		salao.EmailProprietario,
		salao.HashSenha,
		salao.WhatsappNotificacao,
		salao.HorariosFuncionamento, // Por enquanto, pode ser nulo
	).Scan(&salao.ID, &salao.CriadoEm)

	if err != nil {
		// TODO: Checar por erro de violação de constraint UNIQUE (email duplicado)
		log.Printf("Erro ao inserir salão no banco de dados: %v", err)
		http.Error(w, "Erro ao criar salão", http.StatusInternalServerError)
		return
	}

	// 5. Responder com o recurso criado
	salao.HashSenha = "" // Limpamos o hash antes de enviar a resposta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Status 201 Created
	json.NewEncoder(w).Encode(salao)
}
