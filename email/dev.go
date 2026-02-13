package email

import "fmt"

type Dev struct{}

func (Dev) Send(data SendMailData) error {
	fmt.Println("==========")
	fmt.Printf("From: %s <%s>\n", data.FromName, data.FromName)
	fmt.Printf("To: %s\n", data.To)
	fmt.Printf("Subject: %s\n\n", data.Subject)
	fmt.Println(data.Body)
	fmt.Println(data.TextBody)
	fmt.Println(data.HTMLBody)
	fmt.Println("==========")

	return nil
}
