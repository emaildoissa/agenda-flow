package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/emaildoissa/agenda-flow/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	// Carrega as variáveis do arquivo .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Aviso: Erro ao carregar o arquivo .env. Usando variáveis de ambiente do sistema.")
	}

	// Carrega a URL do webhook do n8n
	n8nURL := os.Getenv("N8N_WEBHOOK_URL")
	if n8nURL == "" {
		log.Println("AVISO: A variável de ambiente N8N_WEBHOOK_URL não está definida.")
	}

	// <<< INÍCIO DA MODIFICAÇÃO >>>
	// Carrega as credenciais do banco de dados das variáveis de ambiente
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	// Monta a string de conexão (DSN - Data Source Name)
	dbConnectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	// <<< FIM DA MODIFICAÇÃO >>>

	// Conexão com o banco de dados
	db, err := connectDB(dbConnectionString)
	if err != nil {
		log.Fatalf("Não foi possível conectar ao banco de dados: %v", err)
	}
	defer db.Close()
	log.Println("Conexão com o banco de dados bem-sucedida!")

	// Configuração do roteador e middlewares
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// --- Handlers e Rotas ---
	saloesHandler := handlers.NewSaloesHandler(db)
	r.Post("/saloes", saloesHandler.CreateSalao)
	r.Get("/saloes/{idSalao}", saloesHandler.GetSalaoByID)

	servicosHandler := handlers.NewServicosHandler(db)
	r.Route("/saloes/{idSalao}/servicos", func(r chi.Router) {
		r.Post("/", servicosHandler.CreateServico)
		r.Get("/", servicosHandler.ListServicosBySalaoID)
	})

	agendamentosHandler := handlers.NewAgendamentosHandler(db, n8nURL)
	r.Post("/agendamentos", agendamentosHandler.CreateAgendamento)
	r.Put("/agendamentos/{idAgendamento}/confirmar", agendamentosHandler.ConfirmarAgendamento)
	r.Put("/agendamentos/{idAgendamento}/cancelar", agendamentosHandler.CancelarAgendamento)

	disponibilidadeHandler := handlers.NewDisponibilidadeHandler(db)
	r.Get("/saloes/{idSalao}/disponibilidade", disponibilidadeHandler.GetDisponibilidade)

	r.Get("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	// Iniciar servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor iniciado na porta %s", port)
	err = http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Fatal(err)
	}
}

func connectDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão com o banco: %w", err)
	}
	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("erro ao pingar o banco de dados: %w", err)
	}
	return db, nil
}
