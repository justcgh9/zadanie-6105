package tender

import "time"

type TenderRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ServiceType     string `json:"serviceType"`
	OrganizationId  string `json:"organizationId"`
	CreatorUsername string `json:"creatorUsername"`
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
