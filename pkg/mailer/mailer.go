package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

var Client *Mailer

func InitMailer() {
	Client = &Mailer{
		host:     os.Getenv("MAIL_HOST"),
		port:     os.Getenv("MAIL_PORT"),
		username: os.Getenv("MAIL_USERNAME"),
		password: os.Getenv("MAIL_PASSWORD"),
		from:     os.Getenv("MAIL_FROM"),
	}
	log.Println("[mailer] client initialized")
}

func (m *Mailer) SendOTP(toEmail string, otp string) error {
	subject := "UT StudyPal - Kode Verifikasi OTP"
	body := fmt.Sprintf(`
Halo,

Kode OTP kamu adalah:

  %s

Kode ini berlaku selama 10 menit. Jangan bagikan ke siapapun.

Salam,
UT StudyPal
`, otp)

	return m.send(toEmail, subject, body)
}

func (m *Mailer) send(to, subject, body string) error {
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body,
	)

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
