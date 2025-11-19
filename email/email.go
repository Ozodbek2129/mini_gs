package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/smtp"
	"strconv"
	"time"
)

func Email(email string) (string, error) {
	source := rand.NewSource(time.Now().UnixNano())
	myRand := rand.New(source)

	randomNumber := myRand.Intn(900000) + 100000
	code := strconv.Itoa(randomNumber)

	err := SendCode(email, code)
	if err != nil {
		return "", err
	}

	return "Sizning emailingizga xabar yuborildi", nil
}

func SendCode(email string, code string) error {
	from := "articanconnection@gmail.com"
	password := "inzr pnmv twtv tfbo"

	smtpHost := "smtp.gmail.com"
	smtpPort := "465"

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsConfig)
	if err != nil {
		log.Printf("TLS ulanish xatosi: %v", err)
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Printf("SMTP klient yaratish xatosi: %v", err)
		return err
	}
	defer client.Close()

	auth := smtp.PlainAuth("", from, password, smtpHost)
	if err := client.Auth(auth); err != nil {
		log.Printf("SMTP autentifikatsiya xatosi: %v", err)
		return err
	}

	if err := client.Mail(from); err != nil {
		log.Printf("Sender belgilash xatosi: %v", err)
		return err
	}
	if err := client.Rcpt(email); err != nil {
		log.Printf("Qabul qiluvchi belgilash xatosi: %v", err)
		return err
	}

	writer, err := client.Data()
	if err != nil {
		log.Printf("Data yozish xatosi: %v", err)
		return err
	}
	defer writer.Close()

	t, err := template.ParseFiles("email/template.html")
	if err != nil {
		log.Printf("Templateni o‘qish xatosi: %v", err)
		return err
	}

	var body bytes.Buffer
	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: Sizning tasdiqlash kodingiz\n%s\n\n", mimeHeaders)))

	err = t.Execute(&body, struct {
		Passwd string
	}{
		Passwd: code,
	})
	if err != nil {
		log.Printf("Templateni bajarish xatosi: %v", err)
		return err
	}

	if _, err := writer.Write(body.Bytes()); err != nil {
		log.Printf("Xabarni yozish xatosi: %v", err)
		return err
	}

	log.Println("Xabar muvaffaqiyatli yuborildi")
	return nil
}
