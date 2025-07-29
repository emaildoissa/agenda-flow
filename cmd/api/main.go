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
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// TODO: Mover para variáveis de ambiente em produção
	dbConnectionString := "user=postgres password=vcdmsa77 dbname=agenda_flow sslmode=disable"

	db, err := connectDB(dbConnectionString)
	if err != nil {
		log.Fatalf("Não foi possível conectar ao banco de dados: %v", err)
	}
	defer db.Close()

	log.Println("Conexão com o banco de dados bem-sucedida!")

	r := chi.NewRouter()
	r.Use(middleware.Logger)    // Middleware para logar as requisições
	r.Use(middleware.Recoverer) // Middleware para recuperar de panics

	servicosHandler := handlers.NewServicosHandler(db)

	agendamentosHandler := handlers.NewAgendamentosHandler(db)

	// Rota pública para criar uma solicitação de agendamento
	r.Post("/agendamentos", agendamentosHandler.CreateAgendamento)

	// Rotas "internas" para o n8n usar
	r.Put("/agendamentos/{idAgendamento}/confirmar", agendamentosHandler.ConfirmarAgendamento)
	r.Put("/agendamentos/{idAgendamento}/cancelar", agendamentosHandler.CancelarAgendamento)

	// Agrupar rotas que pertencem a um salão específico
	r.Route("/saloes/{idSalao}/servicos", func(r chi.Router) {
		r.Post("/", servicosHandler.CreateServico)        // POST /saloes/{idSalao}/servicos
		r.Get("/", servicosHandler.ListServicosBySalaoID) // GET /saloes/{idSalao}/servicos
	})

	saloesHandler := handlers.NewSaloesHandler(db)

	r.Post("/saloes", saloesHandler.CreateSalao)

	r.Get("/saloes/{idSalao}", saloesHandler.GetSalaoByID) // Método para buscar um salão pelo ID

	r.Get("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

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
