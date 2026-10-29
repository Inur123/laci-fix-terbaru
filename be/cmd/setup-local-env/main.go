package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	source := flag.String("source", "", "path to the original Next.js .env")
	database := flag.String("database", "", "connection URL for the restored local database")
	flag.Parse()
	if *source == "" || *database == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/setup-local-env -source /path/to/old/.env -database postgresql://...")
		os.Exit(2)
	}

	values, err := godotenv.Read(*source)
	if err != nil {
		fail(err)
	}
	key := values["ENCRYPTION_KEY"]
	if key == "" {
		fail(errors.New("ENCRYPTION_KEY is missing in the original .env"))
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		fail(err)
	}

	f, err := os.OpenFile(".env", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fail(err)
	}
	_, err = fmt.Fprintf(f, "APP_ENV=development\nHTTP_ADDR=127.0.0.1:8080\nFRONTEND_URL=http://localhost:3000\nDATABASE_URL=%q\nENCRYPTION_KEY=%q\nSESSION_SECRET=%s\n", *database, key, hex.EncodeToString(secret))
	if err != nil {
		_ = f.Close()
		_ = os.Remove(".env")
		fail(err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(".env")
		fail(err)
	}
	fmt.Println("Local backend .env created; external mail, Google, and R2 credentials were not copied.")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
