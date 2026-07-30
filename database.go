package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type ServicoData struct {
	Nome             string `json:"nome"`
	ConvencionalQtd  int    `json:"convencional_qtd"`
	PrioritarioQtd   int    `json:"prioritario_qtd"`
	TempoEsperaMedio int    `json:"tempo_espera_medio"`
	Fluxo            int    `json:"fluxo"`
}

type Payload struct {
	UnidadeID string        `json:"unidade_id"`
	Servicos  []ServicoData `json:"servicos"`
}

type Db struct {
	Cfg Config
}

func (db *Db) Connect() *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", db.Cfg.DB_USER, db.Cfg.DB_PASS, db.Cfg.DB_HOST, db.Cfg.DB_PORT, db.Cfg.DB_NAME)
	conn, err := sql.Open("mysql", dsn)

	if err != nil {
		log.Fatal("erro ao conectar ao banco", err)
	}

	return conn
}

func fetchData(conn *sql.DB, cfg Config) *Payload {
	rows, err := conn.Query(`
		SELECT s.id, s.nome
		FROM servicos s
		JOIN servicos_unidades su ON su.servico_id = s.id
		WHERE su.unidade_id = ?
		AND su.ativo = 1
		AND s.ativo = 1
		ORDER BY s.nome
	`, cfg.UNIDADE_LOCAL_ID)

	if err != nil {
		log.Println(err)

		return nil
	}

	defer rows.Close()

	var servicos []ServicoData

	for rows.Next() {
		var servicoID int
		var nome string
		if err := rows.Scan(&servicoID, &nome); err != nil {
			log.Printf("Erro ao acessar serviço %d", err)
			return nil
		}

		data, err := calcularStatus(conn, cfg.UNIDADE_LOCAL_ID, servicoID, nome)
		if err != nil {
			log.Printf("Erro ao calcular stats para serviço %d (%s): %v", servicoID, nome, err)
			continue
		}

		data.Nome = nome
		servicos = append(servicos, *data)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erro ao iterar sob serviços %d", err)
		return nil
	}

	if len(servicos) == 0 {
		log.Println("Nenhum serviço ativo encontrado para esta unidade.")
		return nil
	}
	return &Payload{UnidadeID: cfg.UNIDADE_UUID, Servicos: servicos}
}

func (db *Db) SendData(conn *sql.DB, cfg Config) error {
	payload := fetchData(conn, cfg)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))

	// req, err := http.NewRequest("POST", ServerURL, bytes.NewBuffer(jsonData))
	// if err != nil {
	// 	return err
	// }
	// req.Header.Set("Content-Type", "application/json")

	// client := &http.Client{Timeout: 30 * time.Second}
	// resp, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }
	// defer resp.Body.Close()

	// if resp.StatusCode != http.StatusOK {
	// 	body, _ := io.ReadAll(resp.Body)
	// 	return fmt.Errorf("erro na resposta do servidor: %d %s - %s", resp.StatusCode, resp.Status, string(body))
	// }
	return nil
}

func calcularStatus(db *sql.DB, unidadeID int, servicoID int, servicoNome string) (*ServicoData, error) {
	data := &ServicoData{Nome: servicoNome}

	err := db.QueryRow(`
		SELECT 
		COALESCE(SUM(CASE WHEN p.peso > 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN p.peso = 0 THEN 1 ELSE 0 END), 0)
		FROM view_atendimentos v
        INNER JOIN prioridades p ON p.id = v.prioridade_id
		WHERE v.unidade_id = ? 
		AND v.servico_id = ?
		AND v.status = 'emitida'
	`, unidadeID, servicoID).Scan(&data.PrioritarioQtd, &data.ConvencionalQtd)

	if err != nil {
		return nil, err
	}

	totalFila := data.PrioritarioQtd + data.ConvencionalQtd
	var fluxoSegundos float64

	err = db.QueryRow(`
		SELECT 
			IFNULL(AVG(TIMESTAMPDIFF(SECOND, dt_cheg, dt_cha)), 0)
		FROM 
			view_atendimentos
		WHERE 
			unidade_id = ?
			AND servico_id = ? 
			AND dt_cha IS NOT NULL 
			AND dt_cheg IS NOT NULL
			AND dt_cha >= DATE_SUB(NOW(), INTERVAL 12 HOUR)
	`, unidadeID, servicoID).Scan(&fluxoSegundos)

	if err != nil {
		return nil, err
	}

	fluxoMinutos := int((fluxoSegundos + 29) / 60) // arredonda para cima
	data.Fluxo = fluxoMinutos
	data.TempoEsperaMedio = totalFila * fluxoMinutos

	return data, nil
}
