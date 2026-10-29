package httpapi

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"laci-backend/internal/config"
)

func TestSendSMTPLocalDelivery(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	delivered := make(chan string, 1)
	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		writer := bufio.NewWriter(conn)
		reader := bufio.NewScanner(conn)
		_, _ = fmt.Fprint(writer, "220 local.test SMTP ready\r\n")
		_ = writer.Flush()
		inData := false
		var message strings.Builder
		for reader.Scan() {
			line := reader.Text()
			response := "250 OK\r\n"
			switch {
			case inData && line == ".":
				inData = false
				delivered <- message.String()
			case inData:
				message.WriteString(line)
				message.WriteByte('\n')
				continue
			case strings.HasPrefix(line, "EHLO "), strings.HasPrefix(line, "HELO "):
				response = "250 local.test\r\n"
			case line == "DATA":
				response = "354 End data with <CR><LF>.<CR><LF>\r\n"
				inData = true
			case line == "QUIT":
				response = "221 Bye\r\n"
				_, _ = fmt.Fprint(writer, response)
				_ = writer.Flush()
				serverErr <- nil
				return
			}
			_, _ = fmt.Fprint(writer, response)
			if err := writer.Flush(); err != nil {
				serverErr <- err
				return
			}
		}
		serverErr <- reader.Err()
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	message := []byte("Subject: QA\r\n\r\nHello from Laci")
	if err := sendSMTP(ctx, "127.0.0.1", port, "", "", "sender@example.test", "receiver@example.test", message); err != nil {
		t.Fatalf("local SMTP delivery failed: %v", err)
	}
	select {
	case got := <-delivered:
		if !strings.Contains(got, "Hello from Laci") || !strings.Contains(got, "Subject: QA") {
			t.Fatalf("SMTP message incomplete: %q", got)
		}
	case <-ctx.Done():
		t.Fatal("local SMTP server did not receive message")
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestLiveSMTPAuthenticatesWithoutSendingMail(t *testing.T) {
	if os.Getenv("LACI_RUN_EXTERNAL_TESTS") != "1" {
		t.Skip("set LACI_RUN_EXTERNAL_TESTS=1 to check the configured SMTP server")
	}
	_ = godotenv.Load("../../.env")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPHost == "" || cfg.SMTPUser == "" || cfg.SMTPPass == "" {
		t.Skip("SMTP credentials are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(cfg.SMTPHost, fmt.Sprint(cfg.SMTPPort)))
	if err != nil {
		t.Fatalf("SMTP connection failed: %v", err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPPort == 465 {
		secure := tls.Client(connection, &tls.Config{ServerName: cfg.SMTPHost, MinVersion: tls.VersionTLS12})
		if err := secure.HandshakeContext(ctx); err != nil {
			t.Fatalf("SMTP TLS handshake failed: %v", err)
		}
		connection = secure
	}
	client, err := smtp.NewClient(connection, cfg.SMTPHost)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if cfg.SMTPPort != 465 {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			t.Fatal("SMTP server does not offer STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: cfg.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
			t.Fatalf("SMTP STARTTLS failed: %v", err)
		}
	}
	if err := client.Auth(smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)); err != nil {
		t.Fatalf("SMTP authentication failed: %v", err)
	}
	_ = client.Quit()
}

func TestSMTPPlaintextExceptionIsLoopbackOnly(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "::1"} {
		if !isLoopbackSMTPHost(host) {
			t.Errorf("local SMTP host rejected: %q", host)
		}
	}
	for _, host := range []string{"smtp.example.test", "192.0.2.1", "10.0.0.1"} {
		if isLoopbackSMTPHost(host) {
			t.Errorf("remote SMTP host accepted without TLS: %q", host)
		}
	}
}

func TestSendSMTPCancelsWhileWaitingForGreeting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		close(accepted)
		var one [1]byte
		_, _ = conn.Read(one[:])
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-accepted
		cancel()
	}()
	started := time.Now()
	port := listener.Addr().(*net.TCPAddr).Port
	err = sendSMTP(ctx, "127.0.0.1", port, "", "", "sender@example.test", "receiver@example.test", []byte("test"))
	if err == nil || time.Since(started) > time.Second {
		t.Fatalf("canceled SMTP returned error=%v after %s", err, time.Since(started))
	}
	select {
	case <-serverDone:
	case <-time.After(time.Second):
		t.Fatal("local SMTP server goroutine did not exit")
	}
}

func TestPgDumpConnectionKeepsPasswordOutOfArguments(t *testing.T) {
	tests := []struct {
		input, password string
	}{
		{"postgres://user:p%40ssword@127.0.0.1:5432/laci?sslmode=require", "p@ssword"},
		{"postgresql://user@127.0.0.1/laci?password=secret%2Fvalue&sslmode=require", "secret/value"},
	}
	for _, tt := range tests {
		clean, password, err := pgDumpConnection(tt.input)
		if err != nil || password != tt.password || strings.Contains(clean, tt.password) || strings.Contains(clean, "password=") {
			t.Fatalf("unsafe backup connection: clean=%q error=%v", clean, err)
		}
	}
	if _, _, err := pgDumpConnection("host=localhost password=secret"); err == nil {
		t.Fatal("non-URL database DSN must not be passed to pg_dump arguments")
	}
}

func TestPgDumpConnectsWithoutPasswordInArguments(t *testing.T) {
	if os.Getenv("LACI_RUN_DB_TESTS") != "1" {
		t.Skip("set LACI_RUN_DB_TESTS=1 for a read-only pg_dump connection check")
	}
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	clean, password, err := pgDumpConnection(cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pg_dump", "--dbname="+clean, "--schema-only", "--no-owner", "--file="+os.DevNull)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "PGPASSWORD=" + password}
	if err := cmd.Run(); err != nil {
		t.Fatalf("pg_dump could not connect with sanitized arguments: %v", err)
	}
}
