package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DB_USER          string
	DB_NAME          string
	DB_PASS          string
	DB_HOST          string
	DB_PORT          string
	UNIDADE_UUID     string
	UNIDADE_LOCAL_ID int
}

func main() {
	_ = godotenv.Load()

	localID, _ := strconv.Atoi(os.Getenv("UNIDADE_LOCAL_ID"))
	cfg := Config{
		DB_USER:          os.Getenv("DB_USER"),
		DB_NAME:          os.Getenv("DB_NAME"),
		DB_PASS:          os.Getenv("DB_PASS"),
		DB_HOST:          os.Getenv("DB_HOST"),
		DB_PORT:          os.Getenv("DB_PORT"),
		UNIDADE_UUID:     os.Getenv("UNIDADE_UUID"),
		UNIDADE_LOCAL_ID: localID,
	}

	db := Db{Cfg: cfg}
	conn := db.Connect()
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		log.Fatal("Erro ao pingar DB:", err)
	}

	ticker := time.NewTicker(time.Duration(1) * time.Minute)

	for ; true; <-ticker.C {
		err := db.SendData(conn, cfg)

		if err != nil {
			log.Println("Erro no ciclo: ", err)
		} else {
			log.Println("Dados enviados com sucesso")
		}
	}
	db.SendData(conn, cfg)

}
