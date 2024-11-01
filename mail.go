package main

import (
	"fmt"
	"net/smtp"
	"os"

	godotenv "github.com/joho/godotenv"
)

func sendMail(email string) {
	// Gmail SMTP server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587" // TLS port for Gmail

	// Sender and recipient information.
	godotenv.Load(".env.local")

	sender := "osso.online.site@gmail.com"
	password := os.Getenv("APP_PASSWORD") // App Password if using 2FA
	recipient := email

	// Email content.
	subject := "Subject: Test Email from Osso Online\n"
	body := "This is a test email sent from the Osso Online Email Microservice, written in Go"
	message := []byte(subject + "\n" + body)

	// Set up authentication information.
	auth := smtp.PlainAuth("", sender, password, smtpHost)

	// Send email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, sender, []string{recipient}, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return
	}
	fmt.Println("Email sent successfully!")
}
