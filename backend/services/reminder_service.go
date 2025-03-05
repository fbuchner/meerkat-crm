package services

import (
	"fmt"
	"os"
	"perema/models"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gorm.io/gorm"
)

func SendBirthdayReminders(db *gorm.DB) error {
	var contacts []models.Contact
	if err := db.Where("birthday = ?", time.Now().Format("2006-01-02")).Find(&contacts).Error; err != nil {
		return fmt.Errorf("failed to query contacts: %w", err)
	}

	for _, contact := range contacts {
		age := "unknown age"
		zeroTime := time.Time{}

		contactBirthday, validBirthday := contact.Birthday.ToTime()
		if validBirthday && !contactBirthday.Equal(zeroTime) {
			age = fmt.Sprintf("%d years old", time.Now().Year()-contact.Birthday.Time.Year())
		}

		nickname := contact.Nickname
		if nickname == "" {
			nickname = contact.Firstname
		}

		if err := sendBirthdayMail(nickname, contact.Firstname+" "+contact.Lastname, age); err != nil {
			return fmt.Errorf("failed to send email for %s: %w", contact.Firstname, err)
		}
	}
	return nil
}

// We are using Twillio Sendgrid to send e-mails. The free tier allows for up to 100 mails per day.
func sendBirthdayMail(birthday_person_nick, birthday_person, birthday_age string) error {
	toEmail := mail.NewEmail("", os.Getenv("SENDGRID_TO_EMAIL"))
	message := mail.NewV3Mail()
	message.SetTemplateID(os.Getenv("SENDGRID_BIRTHDAY_TEMPLATE_ID"))

	personalization := mail.NewPersonalization()
	personalization.AddTos(toEmail)

	personalization.SetDynamicTemplateData("birthday_person_nick", birthday_person_nick)
	personalization.SetDynamicTemplateData("birthday_person", birthday_person)
	personalization.SetDynamicTemplateData("birthday_age", birthday_age)

	message.AddPersonalizations(personalization)

	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		return err
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}

	return nil
}
