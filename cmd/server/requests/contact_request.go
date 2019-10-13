package requests

import (
	"time"

	"github.com/vovainside/vobook/database/models"
)

type CreateContact struct {
	Name       string                  `json:"name"`
	FirstName  string                  `json:"first_name"`
	LastName   string                  `json:"last_name"`
	MiddleName string                  `json:"middle_name"`
	Birthday   time.Time               `json:"birthday"`
	Properties []CreateContactProperty `json:"properties"`
}

func (r *CreateContact) Validate() (err error) {
	for _, v := range r.Properties {
		err = v.Validate()
		if err != nil {
			return
		}
	}
	return
}

func (r *CreateContact) ToModel() *models.Contact {
	m := &models.Contact{
		Name:       r.Name,
		FirstName:  r.FirstName,
		LastName:   r.LastName,
		MiddleName: r.MiddleName,
		Birthday:   r.Birthday,
	}

	m.Props = make([]models.ContactProperty, len(r.Properties))
	for i, v := range r.Properties {
		m.Props[i] = v.ToModel()
		m.Props[i].Order = i + 1
	}

	return m
}

type UpdateContact struct {
	Name       *string    `json:"name"`
	FirstName  *string    `json:"first_name"`
	LastName   *string    `json:"last_name"`
	MiddleName *string    `json:"middle_name"`
	Birthday   *time.Time `json:"birthday"`
}

func (r *UpdateContact) Validate() (err error) {
	return
}

func (r *UpdateContact) ToModel(m *models.Contact) {
	if r.Name != nil {
		m.Name = *r.Name
	}
	if r.FirstName != nil {
		m.FirstName = *r.FirstName
	}
	if r.LastName != nil {
		m.LastName = *r.LastName
	}
	if r.MiddleName != nil {
		m.MiddleName = *r.MiddleName
	}
	if r.Birthday != nil {
		m.Birthday = *r.Birthday
	}
}

type SearchContact struct {
	Page    int    `form:"page"`
	Limit   int    `form:"per_page"`
	Query   string `form:"query"` // search anything
	Trashed bool   `form:"trashed"`
}

func (r *SearchContact) Validate() (err error) {
	return
}
