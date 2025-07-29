package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/emaildoissa/agenda-flow/internal/models"
	"github.com/go-chi/chi/v5"
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

// GetSalaoByID busca um salão pelo seu ID.
func (h *SaloesHandler) GetSalaoByID(w http.ResponseWriter, r *http.Request) {
	// 1. Pegar o ID da URL. O chi nos ajuda a fazer isso facilmente.
	idStr := chi.URLParam(r, "idSalao")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	// 2. Buscar o salão no banco de dados
	var salao models.Salao
	sqlStatement := `
		SELECT id, nome_salao, email_proprietario, whatsapp_notificacao, horarios_funcionamento, criado_em
		FROM saloes
		WHERE id = $1`

	row := h.DB.QueryRowContext(r.Context(), sqlStatement, id)
	err = row.Scan(
		&salao.ID,
		&salao.NomeSalao,
		&salao.EmailProprietario,
		&salao.WhatsappNotificacao,
		&salao.HorariosFuncionamento,
		&salao.CriadoEm,
	)

	if err != nil {
		// Se o erro for 'sql.ErrNoRows', significa que não encontramos o salão.
		// Retornamos um erro 404 Not Found, que é o correto.
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Salão não encontrado", http.StatusNotFound)
		} else {
			// Para qualquer outro erro, é um problema no servidor.
			log.Printf("Erro ao buscar salão: %v", err)
			http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		}
		return
	}

	// 3. Responder com o JSON do salão encontrado
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Status 200 OK
	json.NewEncoder(w).Encode(salao)
}
