package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/grokify/ringcentral-sdk-go/rcsdk/platform"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	Port                 int
	LogLevel             log.Level
	RcAppKey             string
	RcAppSecret          string
	RcServerURL          string
	RcUsername           string
	RcExtension          string
	RcPassword           string
	RcPhoneNumberRingOut string
	RcPhoneNumberSMS     string
	Platform             platform.Platform
	AddressBook          AddressBook
	Cache                *cache.Cache
}

// Address returns the port address as a string with a `:` prefix
func (c *Configuration) Address() string {
	return fmt.Sprintf(":%d", c.Port)
}

func (c *Configuration) LoadEnv() {
	c.RcAppKey = os.Getenv("RC_APP_KEY")
	c.RcAppSecret = os.Getenv("RC_APP_SECRET")
	c.RcServerURL = os.Getenv("RC_SERVER_URL")
	c.RcUsername = os.Getenv("RC_USERNAME")
	c.RcExtension = os.Getenv("RC_EXTENSION")
	c.RcPassword = os.Getenv("RC_PASSWORD")
	c.RcPhoneNumberRingOut = os.Getenv("RC_PHONE_NUMBER_RINGOUT")
	c.RcPhoneNumberSMS = os.Getenv("RC_PHONE_NUMBER_SMS")
}

func NewConfiguration() (Configuration, error) {
	cfg := Configuration{}
	cfg.LoadEnv()

	addr, err := GetAddressBook(os.Getenv("ADDRESS_BOOK_FILE"))
	if err != nil {
		return cfg, err
	}
	cfg.AddressBook = addr

	sdk, err := GetRingCentralSdk(cfg)
	if err != nil {
		return cfg, err
	}
	cfg.Platform = sdk

	return cfg, nil
}

type AddressBook struct {
	Contacts    []Contact      `json:"contacts,omitempty"`
	ContactsMap map[string]int `json:"-,omitempty"`
}

func (ab *AddressBook) Inflate() {
	contactsMap := map[string]int{}
	for i, contact := range ab.Contacts {
		if len(contact.FirstName) > 0 {
			contactsMap[strings.ToLower(contact.FirstName)] = i
		}
	}
	ab.ContactsMap = contactsMap
}

func (ab *AddressBook) GetContactByFirstName(firstName string) (Contact, error) {
	firstNameLc := strings.ToLower(firstName)
	if idx, ok := ab.ContactsMap[firstNameLc]; ok {
		if len(ab.Contacts) > idx {
			return ab.Contacts[idx], nil
		}
	}
	return Contact{}, errors.New(fmt.Sprintf("Contact Not Found: %v", firstName))
}

type Contact struct {
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
}

func (c *Contact) FullName() string {
	names := []string{}
	if len(strings.TrimSpace(c.FirstName)) > 0 {
		names = append(names, strings.TrimSpace(c.FirstName))
	}
	if len(strings.TrimSpace(c.LastName)) > 0 {
		names = append(names, strings.TrimSpace(c.LastName))
	}
	return strings.Join(names, " ")
}

func GetAddressBook(filepath string) (AddressBook, error) {
	ad := AddressBook{}
	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return ad, err
	}
	err = json.Unmarshal(bytes, &ad)
	return ad, err
}
