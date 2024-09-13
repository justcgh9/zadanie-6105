package postgres

import (
	"database/sql"
	"fmt"
	"tender_system/internal/models/tender"
	"tender_system/internal/models/user"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS tender (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(100) NOT NULL,
		description VARCHAR(500),
		serviceType VARCHAR(50),
		status VARCHAR(50),
		version INT DEFAULT 1,
		createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		organizationId UUID REFERENCES organization(id) ON DELETE CASCADE
	);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = db.Prepare(`
	CREATE TABLE IF NOT EXISTS tenderHistory (
		tenderId UUID,
		name VARCHAR(100) NOT NULL,
		description VARCHAR(500) NOT NULL,
		serviceType VARCHAR(50) NOT NULL,
		status VARCHAR(50) NOT NULL,
		version INT NOT NULL,
		PRIMARY KEY(tenderId, version)
	);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = db.Prepare(`
	CREATE TABLE IF NOT EXISTS bid (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(100) NOT NULL,
		description VARCHAR(500),
		status VARCHAR(50),
		tenderId UUID REFERENCES tender(id) ON DELETE CASCADE,
		authorType VARCHAR(50),
		authorId UUID,
		version INT DEFAULT 1,
		createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = db.Prepare(`
	CREATE TABLE IF NOT EXISTS tenderHolder (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		tenderId UUID REFERENCES tender(id) ON DELETE CASCADE,
		creatorUsername VARCHAR(100) REFERENCES employee(username) ON DELETE CASCADE
	);
	`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveTender(ten tender.TenderRequest) (tender.TenderResponse, error) {
	const op = "storage.postgres.SaveTender"
	var result tender.TenderResponse

	stmt, err := s.db.Prepare(`
	INSERT INTO tender(name, description, serviceType, status, organizationId)
	VALUES ($1, $2, $3, 'Created', $4)
	RETURNING id, name, description, status, serviceType, version, createdAt
	`)

	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(
		ten.Name,
		ten.Description,
		ten.ServiceType,
		ten.OrganizationId,
	).Scan(&result.Id, &result.Name, &result.Description, &result.Status, &result.ServiceType, &result.Version, &result.CreatedAt)

	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO tenderHolder(tenderId, creatorUsername)
	VALUES ($1, $2)
	`)

	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(result.Id, ten.CreatorUsername)

	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil

}

func (s *Storage) ReadTenders(limit, offset int) ([]tender.TenderResponse, error) {
	const op = "storage.postgres.ReadTenders"
	result := make([]tender.TenderResponse, 0)

	stmt, err := s.db.Prepare(`
	SELECT id, name, description, status, serviceType, version, createdAt
	FROM tender
	LIMIT $1
	OFFSET $2
	`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := stmt.Query(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for rows.Next() {
		var ten tender.TenderResponse

		err := rows.Scan(&ten.Id, &ten.Name, &ten.Description, &ten.Status, &ten.ServiceType, &ten.Version, &ten.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		result = append(result, ten)
	}

	return result, nil
}

func (s *Storage) ReadMyTenders(username string, limit int, offset int) ([]tender.TenderResponse, error) {
	const op = "storage.postgres.ReadMyTenders"
	result := make([]tender.TenderResponse, 0)

	stmt, err := s.db.Prepare(`
	SELECT t.id, name, description, status, serviceType, version, createdAt
	FROM tender t
	INNER JOIN tenderHolder th
	ON th.tenderId = t.id
	WHERE creatorUsername=$1
	LIMIT $2
	OFFSET $3;
	`)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := stmt.Query(username, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for rows.Next() {
		var ten tender.TenderResponse

		err := rows.Scan(&ten.Id, &ten.Name, &ten.Description, &ten.Status, &ten.ServiceType, &ten.Version, &ten.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		result = append(result, ten)
	}

	return result, nil
}

func (s *Storage) ReadTenderStatus(tenderId string) (string, string, error) {
	const op = "storage.postgres.ReadTenderStatus"
	var status, username string

	stmt, err := s.db.Prepare(`
	SELECT status, creatorUsername
	FROM tender t
	INNER JOIN tenderHolder th
	ON t.id = th.tenderId
	WHERE t.id=$1
	`)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(tenderId).Scan(&status, &username)
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	return status, username, nil
}

func (s *Storage) CheckOrganizationResponsible(username, organization_id string) (bool, error) {
	const op = "storage.postgres.CheckOrganizationResponsible"

	stmt, err := s.db.Prepare(`
	SELECT user_id
	FROM employee
	WHERE username=$1
	`)

	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	var user_id string

	err = stmt.QueryRow(username).Scan(&user_id)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	SELECT * 
	FROM organization_responsible
	WHERE organization_id=$1 AND user_id=$2
	`)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	var responsible user.OrganizationResponsible
	err = stmt.QueryRow(organization_id, user_id).Scan(&responsible.Id, &responsible.OrganizationId, &responsible.UserId)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	if responsible.OrganizationId != organization_id || responsible.UserId != user_id {
		return false, nil
	}

	return true, nil
}

func (s *Storage) FetchUser(username string) (user.User, error) {
	const op = "storage.postgres.FetchUser"
	var usr user.User

	stmt, err := s.db.Prepare(`
	SELECT id, username, first_name, last_name, created_at, updated_at
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return user.User{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(username).Scan(&usr.Id, &usr.Username, &usr.FirstName, &usr.LastName, &usr.CreatedAt, &usr.UpdatedAt)
	if err != nil {
		return user.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return usr, nil

}

func (s *Storage) FetchUserOrganization(username string) (string, error) {
	const op = "storage.postgres.FetchUserOrganization"
	var orgId string

	stmt, err := s.db.Prepare(`
	SELECT organization_id
	FROM organization_responsible a
	JOIN employee b
	ON a.user_id = b.id
	WHERE username = $1
	`)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(username).Scan(&orgId)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return orgId, nil
}

func (s *Storage) UpdateTenderStatus(tenderId, status, username string) (tender.TenderResponse, error) {
	const op = "storage.postgres.UpdateTenderStatus"

	stmt, err := s.db.Prepare(`
	SELECT t.id, t.name, t.description, t.serviceType, t.status, t.version, t.createdAt, creatorUsername
	FROM tender t
	INNER JOIN tenderHolder th
	ON t.id = th.tenderId
	WHERE t.id = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var ten tender.TenderResponse
	var creatorUsername string
	err = stmt.QueryRow(tenderId).Scan(&ten.Id, &ten.Name, &ten.Description, &ten.ServiceType, &ten.Status, &ten.Version, &ten.CreatedAt, &creatorUsername)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	if username != creatorUsername {
		return tender.TenderResponse{}, fmt.Errorf("%s: %s", op, "Given User is not a creator")
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO tenderHistory(tenderId, name, description, serviceType, status, version)
	VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(ten.Id, ten.Name, ten.Description, ten.ServiceType, ten.Status, ten.Version)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	UPDATE tender 
	SET status = $1, version = version + 1
	WHERE id = $2
	RETURNING status, version
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(status, ten.Id).Scan(&ten.Status, &ten.Version)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return ten, nil
}

func (s *Storage) PatchTender(tenderId, username, name, description, serviceType string) (tender.TenderResponse, error) {
	const op = "storage.postgres.PatchTender"
	var result tender.TenderResponse

	query := "UPDATE tender SET version = version + 1"
	if name != "" {
		query = fmt.Sprintf("%s, name = '%s'", query, name)
	}
	if description != "" {
		query = fmt.Sprintf("%s, description = '%s'", query, description)
	}
	if serviceType != "" {
		query = fmt.Sprintf("%s, serviceType = '%s'", query, serviceType)
	}
	stmt, err := s.db.Prepare(`
	SELECT t.id, t.name, t.description, t.serviceType, t.status, t.version, t.createdAt, creatorUsername
	FROM tender t
	INNER JOIN tenderHolder th
	ON t.id = th.tenderId
	WHERE t.id = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var ten tender.TenderResponse
	var creatorUsername string
	err = stmt.QueryRow(tenderId).Scan(&ten.Id, &ten.Name, &ten.Description, &ten.ServiceType, &ten.Status, &ten.Version, &ten.CreatedAt, &creatorUsername)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	if username != creatorUsername {
		return tender.TenderResponse{}, fmt.Errorf("%s: %s", op, "Given User is not a creator")
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO tenderHistory(tenderId, name, description, serviceType, status, version)
	VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(ten.Id, ten.Name, ten.Description, ten.ServiceType, ten.Status, ten.Version)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	query = fmt.Sprintf("%s WHERE id = '%s' RETURNING id, name, description, serviceType, status, version, createdAt", query, tenderId)
	stmt, err = s.db.Prepare(query)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow().Scan(&result.Id, &result.Name, &result.Description, &result.ServiceType, &result.Status, &result.Version, &result.CreatedAt)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	return result, nil
}

func (s *Storage) RollbackTender(tenderId, username string, version int) (tender.TenderResponse, error) {
	const op = "storage.postgres.RollbackTender"
	var result tender.TenderResponse
	stmt, err := s.db.Prepare(`
	SELECT t.id, t.name, t.description, t.serviceType, t.status, t.version, t.createdAt, creatorUsername
	FROM tender t
	INNER JOIN tenderHolder th
	ON t.id = th.tenderId
	WHERE t.id = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var ten tender.TenderResponse
	var creatorUsername string
	err = stmt.QueryRow(tenderId).Scan(&ten.Id, &ten.Name, &ten.Description, &ten.ServiceType, &ten.Status, &ten.Version, &ten.CreatedAt, &creatorUsername)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	if username != creatorUsername {
		return tender.TenderResponse{}, fmt.Errorf("%s: %s", op, "Given User is not a creator")
	}

	if version > int(ten.Version) || version <= 0 {
		return tender.TenderResponse{}, fmt.Errorf("%s: %s", op, "Invalid version")
	}

	if version == int(ten.Version) {
		return ten, nil
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO tenderHistory(tenderId, name, description, serviceType, status, version)
	VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(ten.Id, ten.Name, ten.Description, ten.ServiceType, ten.Status, ten.Version)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	SELECT tenderId, name, description, serviceType, status
	FROM tenderHistory
	WHERE version = $1 AND tenderId = $2
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(version, tenderId).Scan(&ten.Id, &ten.Name, &ten.Description, &ten.ServiceType, &ten.Status)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	UPDATE tender
	SET name = $1, description = $2, serviceType = $3, status = $4, version = version + 1
	WHERE id = $5
	RETURNING id, name, description, status, serviceType, version, createdAt
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(ten.Name, ten.Description, ten.ServiceType, ten.Status, tenderId).Scan(&result.Id, &result.Name, &result.Description, &result.Status, &result.ServiceType, &result.Version, &result.CreatedAt)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}
