package tender

import "time"

type TenderRequest struct {
	Name            string `json:"name" validate:"required"`
	Description     string `json:"description" validate:"required"`
	ServiceType     string `json:"serviceType" validate:"required"`
	OrganizationId  string `json:"organizationId" validate:"required"`
	CreatorUsername string `json:"creatorUsername" validate:"required"`
}

type TenderResponse struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ServiceType string    `json:"serviceType"`
	Status      string    `json:"status"`
	Version     int32     `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
}

type TenderPatchRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	ServiceType string `json:"serviceType,omitempty"`
}
