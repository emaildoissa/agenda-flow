-- Habilita a extensão para gerar UUIDs, se quisermos usá-los no futuro
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabela para os salões (nossos clientes/assinantes)
CREATE TABLE saloes (
    id SERIAL PRIMARY KEY,
    nome_salao VARCHAR(255) NOT NULL,
    email_proprietario VARCHAR(255) UNIQUE NOT NULL,
    hash_senha VARCHAR(255) NOT NULL, -- IMPORTANTE: NUNCA guarde senhas em texto plano
    whatsapp_notificacao VARCHAR(20) NOT NULL,
    horarios_funcionamento JSONB, -- JSONB é ótimo para estruturas flexíveis de horários
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabela para os serviços oferecidos por cada salão
CREATE TABLE servicos (
    id SERIAL PRIMARY KEY,
    salao_id INTEGER NOT NULL REFERENCES saloes(id) ON DELETE CASCADE,
    nome VARCHAR(255) NOT NULL,
    duracao_minutos INTEGER NOT NULL,
    preco DECIMAL(10, 2) NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE
);

-- Tabela para os agendamentos
CREATE TABLE agendamentos (
    id SERIAL PRIMARY KEY,
    salao_id INTEGER NOT NULL REFERENCES saloes(id) ON DELETE CASCADE,
    servico_id INTEGER NOT NULL REFERENCES servicos(id),
    cliente_nome VARCHAR(255) NOT NULL,
    cliente_contato VARCHAR(255) NOT NULL,
    data_hora_inicio TIMESTAMPTZ NOT NULL,
    data_hora_fim TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('PENDENTE', 'CONFIRMADO', 'CANCELADO', 'CONCLUIDO')),
    criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Índices para otimizar buscas comuns
CREATE INDEX idx_agendamentos_salao_data ON agendamentos(salao_id, data_hora_inicio);