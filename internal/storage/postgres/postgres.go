package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"tender_system/internal/models/bids"
	"tender_system/internal/models/tender"
	"tender_system/internal/models/user"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

var (
	ErrBadRequest   = errors.New("bad request")
	ErrUserNotFound = errors.New("user doesn't exist or is invalid")
	ErrForbidden    = errors.New("not enough access rights")
	ErrNotFound     = errors.New("404 Not Found")
)

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
	CREATE TABLE IF NOT EXISTS bidHistory (
		bidId UUID NOT NULL,
		name VARCHAR(100) NOT NULL,
		description VARCHAR(500) NOT NULL,
		status VARCHAR(50) NOT NULL,
		version INT NOT NULL,
		PRIMARY KEY(bidId, version)
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

	stmt, err = db.Prepare(`
	CREATE TABLE IF NOT EXISTS feedback (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		bidId UUID REFERENCES bid(id) ON DELETE CASCADE,
		description VARCHAR(1000) NOT NULL,
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
	CREATE TABLE IF NOT EXISTS decisions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		status VARCHAR(15),
		numApproved INT DEFAULT 0,
		bidId UUID REFERENCES bid(id) ON DELETE CASCADE
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
	CREATE TABLE IF NOT EXISTS voted (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		username VARCHAR(100),
		user_id UUID REFERENCES employee(id),
		decision VARCHAR(15),
		bidId UUID REFERENCES bid(id) ON DELETE CASCADE
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

	var user_id string
	stmt, err := s.db.Prepare(`
	SELECT id 
	FROM employee
	WHERE username = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		stmt.Close()
	}()

	err = stmt.QueryRow(ten.CreatorUsername).Scan(&user_id)
	if err != nil {
		return tender.TenderResponse{}, ErrUserNotFound
	}

	var trash int
	stmt, err = s.db.Prepare(`
	SELECT 1 
	FROM organization
	WHERE id = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(ten.OrganizationId).Scan(&trash)
	if err != nil {
		return tender.TenderResponse{}, ErrBadRequest
	}

	stmt, err = s.db.Prepare(`
	SELECT 1
	FROM organization_responsible
	WHERE organization_id=$1 AND user_id=$2
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	err = stmt.QueryRow(ten.OrganizationId, user_id).Scan(&trash)
	if err != nil {
		return tender.TenderResponse{}, ErrForbidden
	}

	var result tender.TenderResponse

	stmt, err = s.db.Prepare(`
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

func (s *Storage) ReadTenders(limit, offset int, serviceType string) ([]tender.TenderResponse, error) {
	const op = "storage.postgres.ReadTenders"
	result := make([]tender.TenderResponse, 0)
	var query string
	if serviceType == "" {
		query = `
		SELECT id, name, description, status, serviceType, version, createdAt
		FROM tender
		LIMIT $1
		OFFSET $2
		`
	} else {
		query = fmt.Sprintf(`
	SELECT id, name, description, status, serviceType, version, createdAt
	FROM tender
	WHERE serviceType='%s'
	LIMIT $1
	OFFSET $2
	`, serviceType)
	}
	stmt, err := s.db.Prepare(query)

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
	var user_id string
	stmt, err := s.db.Prepare(`
	SELECT id 
	FROM employee
	WHERE username = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		stmt.Close()
	}()

	err = stmt.QueryRow(username).Scan(&user_id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	stmt, err = s.db.Prepare(`
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

func (s *Storage) ReadTenderStatus(tenderId, username string) (string, error) {
	const op = "storage.postgres.ReadTenderStatus"
	var status, organization_id string

	stmt, err := s.db.Prepare(`
	SELECT status, organizationId
	FROM tender 
	WHERE id=$1
	`)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(tenderId).Scan(&status, &organization_id)
	if err != nil {
		return "", ErrNotFound
	}

	if status == "Published" {
		return status, nil
	}

	var user_id string
	stmt, err = s.db.Prepare(`
	SELECT id 
	FROM employee
	WHERE username = $1
	`)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		stmt.Close()
	}()

	err = stmt.QueryRow(username).Scan(&user_id)
	if err != nil {
		return "", ErrUserNotFound
	}

	var trash int

	stmt, err = s.db.Prepare(`
	SELECT 1
	FROM organization_responsible
	WHERE organization_id=$1 AND user_id=$2
	`)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	err = stmt.QueryRow(organization_id, user_id).Scan(&trash)
	if err != nil {
		return "", ErrForbidden
	}

	return status, nil
}

func (s *Storage) CheckOrganizationResponsible(username, organization_id string) (bool, error) {
	const op = "storage.postgres.CheckOrganizationResponsible"

	stmt, err := s.db.Prepare(`
	SELECT id
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
	SELECT id, name, description, serviceType, status, version, createdAt, organizationId
	FROM tender 
	WHERE id = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var ten tender.TenderResponse
	var organization_id string
	err = stmt.QueryRow(tenderId).Scan(&ten.Id, &ten.Name, &ten.Description, &ten.ServiceType, &ten.Status, &ten.Version, &ten.CreatedAt, &organization_id)
	if err != nil {
		return tender.TenderResponse{}, ErrNotFound
	}

	var user_id string
	stmt, err = s.db.Prepare(`
	SELECT id 
	FROM employee
	WHERE username = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		stmt.Close()
	}()

	err = stmt.QueryRow(username).Scan(&user_id)
	if err != nil {
		return tender.TenderResponse{}, ErrUserNotFound
	}

	var trash int

	stmt, err = s.db.Prepare(`
	SELECT 1
	FROM organization_responsible
	WHERE organization_id=$1 AND user_id=$2
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	err = stmt.QueryRow(organization_id, user_id).Scan(&trash)
	if err != nil {
		return tender.TenderResponse{}, ErrForbidden
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
	SELECT t.id, t.name, t.description, t.serviceType, t.status, t.version, t.createdAt, organizationId
	FROM tender t
	INNER JOIN tenderHolder th
	ON t.id = th.tenderId
	WHERE t.id = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var ten tender.TenderResponse
	var organization_id string
	err = stmt.QueryRow(tenderId).Scan(&ten.Id, &ten.Name, &ten.Description, &ten.ServiceType, &ten.Status, &ten.Version, &ten.CreatedAt, &organization_id)
	if err != nil {
		return tender.TenderResponse{}, ErrNotFound
	}
	var user_id string
	stmt, err = s.db.Prepare(`
	SELECT id 
	FROM employee
	WHERE username = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		stmt.Close()
	}()

	err = stmt.QueryRow(username).Scan(&user_id)
	if err != nil {
		return tender.TenderResponse{}, ErrUserNotFound
	}

	var trash int

	stmt, err = s.db.Prepare(`
	SELECT 1
	FROM organization_responsible
	WHERE organization_id=$1 AND user_id=$2
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	err = stmt.QueryRow(organization_id, user_id).Scan(&trash)
	if err != nil {
		return tender.TenderResponse{}, ErrForbidden
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
	SELECT t.id, t.name, t.description, t.serviceType, t.status, t.version, t.createdAt, organizationId
	FROM tender t
	INNER JOIN tenderHolder th
	ON t.id = th.tenderId
	WHERE t.id = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var ten tender.TenderResponse
	var organization_id string
	err = stmt.QueryRow(tenderId).Scan(&ten.Id, &ten.Name, &ten.Description, &ten.ServiceType, &ten.Status, &ten.Version, &ten.CreatedAt, &organization_id)
	if err != nil {
		return tender.TenderResponse{}, ErrNotFound
	}
	var user_id string
	stmt, err = s.db.Prepare(`
	SELECT id 
	FROM employee
	WHERE username = $1
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		stmt.Close()
	}()

	err = stmt.QueryRow(username).Scan(&user_id)
	if err != nil {
		return tender.TenderResponse{}, ErrUserNotFound
	}

	var trash int

	stmt, err = s.db.Prepare(`
	SELECT 1
	FROM organization_responsible
	WHERE organization_id=$1 AND user_id=$2
	`)
	if err != nil {
		return tender.TenderResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	err = stmt.QueryRow(organization_id, user_id).Scan(&trash)
	if err != nil {
		return tender.TenderResponse{}, ErrForbidden
	}

	if version > int(ten.Version) || version <= 0 {
		return tender.TenderResponse{}, ErrBadRequest
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

func (s *Storage) SaveBid(bid bids.BidRequest) (bids.BidResponse, error) {
	const op = "storage.postgres.SaveBid"
	var resp bids.BidResponse
	var uuid, query string

	if bid.AuthorType == "Organization" {
		query = `SELECT id FROM organization WHERE id = $1`
	} else {
		query = `SELECT username FROM employee WHERE id = $1`
	}

	stmt, err := s.db.Prepare(query)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bid.AuthorId).Scan(&uuid)
	if err != nil {
		return bids.BidResponse{}, ErrUserNotFound
	}

	if bid.AuthorType == "User" {
		_, err = s.FetchUserOrganization(uuid)
		if err != nil {
			return bids.BidResponse{}, ErrForbidden
		}
	}

	stmt, err = s.db.Prepare(`
	SELECT status 
	FROM tender
	WHERE id = $1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	var trash string
	err = stmt.QueryRow(bid.TenderId).Scan(&trash)
	if err != nil {
		return bids.BidResponse{}, ErrNotFound
	}
	if trash == "Closed" {
		return bids.BidResponse{}, fmt.Errorf("the tender is closed")
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO bid(name, description, status, tenderId, authorType, authorId)
	VALUES ($1, $2, 'Created', $3, $4, $5)
	RETURNING id, name, status, authorType, authorId, version, createdAt
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(
		bid.Name,
		bid.Description,
		bid.TenderId,
		bid.AuthorType,
		bid.AuthorId,
	).Scan(
		&resp.Id,
		&resp.Name,
		&resp.Status,
		&resp.AuthorType,
		&resp.AuthorId,
		&resp.Version,
		&resp.CreatedAt,
	)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

func (s *Storage) ReadMyBids(username string, limit int, offset int) ([]bids.BidResponse, error) {
	const op = "storage.postgres.ReadMyBids"
	var uuid string
	var resp = make([]bids.BidResponse, 0)

	stmt, err := s.db.Prepare(`
	SELECT id 
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(username).Scan(&uuid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT id, name, status, authorType, authorId, version, createdAt
	FROM bid 
	WHERE authorType='User' AND authorId=$1
	LIMIT $2
	OFFSET $3
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := stmt.Query(uuid, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var bid bids.BidResponse
	for rows.Next() {
		err = rows.Scan(
			&bid.Id,
			&bid.Name,
			&bid.Status,
			&bid.AuthorType,
			&bid.AuthorId,
			&bid.Version,
			&bid.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		resp = append(resp, bid)
	}

	return resp, nil
}

func (s *Storage) ReadTenderBids(username string, tenderId string, limit int, offset int) ([]bids.BidResponse, error) {
	const op = "storage.postgres.ReadTenderBids"
	var resp []bids.BidResponse

	stmt, err := s.db.Prepare(`
	SELECT organizationId
	FROM tender
	WHERE id=$1
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var tName string
	err = stmt.QueryRow(tenderId).Scan(&tName)
	if err != nil {
		return nil, ErrNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT id
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var uuid string
	err = stmt.QueryRow(username).Scan(&uuid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT id, name, status, authorType, authorId, version, createdAt
	FROM bid 
	WHERE tenderId=$1
	LIMIT $2
	OFFSET $3
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := stmt.Query(tenderId, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	flag := false
	organization_id, err := s.FetchUserOrganization(username)
	if err != nil {
		flag = true
	}
	var bid bids.BidResponse
	for rows.Next() {
		err = rows.Scan(
			&bid.Id,
			&bid.Name,
			&bid.Status,
			&bid.AuthorType,
			&bid.AuthorId,
			&bid.Version,
			&bid.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		if bid.AuthorId == uuid || bid.AuthorId == organization_id || organization_id == tName {
			resp = append(resp, bid)
			flag = false
		}
	}
	if flag {
		return nil, ErrForbidden
	}

	return resp, nil
}

func (s *Storage) GetBidStatus(bidId, username string) (string, error) {
	const op = "storage.postgres.GetBidStatus"
	var status, authorId, authorType, tenderId string
	stmt, err := s.db.Prepare(`
	SELECT status, authorType, authorId, tenderId
	FROM bid
	WHERE id=$1
	`)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bidId).Scan(&status, &authorType, &authorId, &tenderId)
	if err != nil {
		return "", ErrNotFound
	}

	if authorType == "User" {
		var uuid string
		stmt, err := s.db.Prepare(`
		SELECT id
		FROM employee
		WHERE username=$1
		`)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}

		err = stmt.QueryRow(username).Scan(&uuid)
		if err != nil {
			return "", ErrUserNotFound
		}
		if uuid != authorId {
			var uname string
			stmt, err := s.db.Prepare(`
			SELECT creatorUsername
			FROM tenderHolder
			WHERE tenderId=$1
			`)
			if err != nil {
				return "", fmt.Errorf("%s: %w", op, err)
			}

			err = stmt.QueryRow(tenderId).Scan(&uname)
			if err != nil {
				return "", fmt.Errorf("%s: %w", op, err)
			}
			if uname != username {
				return "", ErrForbidden
			}
		}
	} else {
		stmt, err := s.db.Prepare(`
		SELECT username
		FROM organization_responsible a 
		JOIN employee b ON a.user_id=b.id
		WHERE a.organization_id=$1
		`)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}

		rows, err := stmt.Query(authorId)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
		flag := false
		for rows.Next() {
			var uname string
			err = rows.Scan(&uname)
			if err != nil {
				return "", fmt.Errorf("%s: %w", op, err)
			}
			if uname == username {
				flag = true
				break
			}
		}
		if !flag {
			var uname string
			stmt, err := s.db.Prepare(`
			SELECT creatorUsername
			FROM tenderHolder
			WHERE tenderId=$1
			`)
			if err != nil {
				return "", fmt.Errorf("%s: %w", op, err)
			}

			err = stmt.QueryRow(tenderId).Scan(&uname)
			if err != nil {
				return "", fmt.Errorf("%s: %w", op, err)
			}
			if uname != username {
				return "", ErrForbidden
			}
		}
	}

	return status, nil
}

func (s *Storage) ChangeBidStatus(bidId, status, username string) (bids.BidResponse, error) {
	const op = "storage.postgres.ChangeBidStatus"
	var bid bids.BidResponse
	var tenderId, description string
	stmt, err := s.db.Prepare(`
	SELECT id, name, status, authorType, authorId, version, createdAt, tenderId, description
	FROM bid
	WHERE id=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bidId).Scan(&bid.Id, &bid.Name, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt, &tenderId, &description)
	if err != nil {
		return bids.BidResponse{}, ErrNotFound
	}

	if bid.AuthorType == "User" {
		var uuid string
		stmt, err := s.db.Prepare(`
		SELECT id
		FROM employee
		WHERE username=$1
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		err = stmt.QueryRow(username).Scan(&uuid)
		if err != nil {
			return bids.BidResponse{}, ErrUserNotFound
		}
		if uuid != bid.AuthorId {
			var uname1, uname2 string
			stmt, err := s.db.Prepare(`
			select e1.user_id as e1_id, e2.user_id as e2_id 
			from organization_responsible e1 
			join organization_responsible e2 
			on e1.organization_id = e2.organization_id
				and e1.id != e2.id 
			where e1.user_id=$1 and e2.user_id=$2
			`)
			if err != nil {
				return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
			}

			err = stmt.QueryRow(bid.AuthorId, uuid).Scan(&uname1, &uname2)
			if err != nil {
				return bids.BidResponse{}, ErrForbidden
			}
		}
	} else {
		stmt, err := s.db.Prepare(`
		SELECT username
		FROM organization_responsible a 
		JOIN employee b ON a.user_id=b.id
		WHERE a.organization_id=$1
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		rows, err := stmt.Query(bid.AuthorId)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}
		flag := false
		for rows.Next() {
			var uname string
			err = rows.Scan(&uname)
			if err != nil {
				return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
			}
			if uname == username {
				flag = true
				break
			}
		}
		if !flag {
			return bids.BidResponse{}, ErrForbidden
		}
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO bidHistory(bidId, name, description, status, version)
	VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(bid.Id, bid.Name, description, bid.Status, bid.Version)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	UPDATE bid
	SET version = version + 1, status=$1
	WHERE id=$2
	RETURNING id, name, status, authorType, authorId, version, createdAt
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(status, bidId).Scan(&bid.Id, &bid.Name, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return bid, nil
}

func (s *Storage) EditBid(bidId, username, name, desc string) (bids.BidResponse, error) {
	const op = "storage.postgres.EditBid"
	var bid bids.BidResponse
	var tenderId, description string
	stmt, err := s.db.Prepare(`
	SELECT id, name, status, authorType, authorId, version, createdAt, tenderId, description
	FROM bid
	WHERE id=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bidId).Scan(&bid.Id, &bid.Name, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt, &tenderId, &description)
	if err != nil {
		return bids.BidResponse{}, ErrNotFound
	}

	if bid.AuthorType == "User" {
		var uuid string
		stmt, err := s.db.Prepare(`
		SELECT id
		FROM employee
		WHERE username=$1
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		err = stmt.QueryRow(username).Scan(&uuid)
		if err != nil {
			return bids.BidResponse{}, ErrUserNotFound
		}
		if uuid != bid.AuthorId {
			var uname1, uname2 string
			stmt, err := s.db.Prepare(`
			select e1.user_id as e1_id, e2.user_id as e2_id 
			from organization_responsible e1 
			join organization_responsible e2 
			on e1.organization_id = e2.organization_id
				and e1.id != e2.id 
			where e1.user_id=$1 and e2.user_id=$2
			`)
			if err != nil {
				return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
			}

			err = stmt.QueryRow(bid.AuthorId, uuid).Scan(&uname1, &uname2)
			if err != nil {
				return bids.BidResponse{}, ErrForbidden
			}
		}

	} else {
		stmt, err := s.db.Prepare(`
		SELECT username
		FROM organization_responsible a 
		JOIN employee b ON a.user_id=b.id
		WHERE a.organization_id=$1
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		rows, err := stmt.Query(bid.AuthorId)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}
		flag := false
		for rows.Next() {
			var uname string
			err = rows.Scan(&uname)
			if err != nil {
				return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
			}
			if uname == username {
				flag = true
				break
			}
		}
		if !flag {
			return bids.BidResponse{}, ErrForbidden
		}
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO bidHistory(bidId, name, description, status, version)
	VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(bid.Id, bid.Name, description, bid.Status, bid.Version)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	query := `
	UPDATE bid
	SET version = version + 1
	`
	if desc != "" {
		query = fmt.Sprintf("%s, description = '%s'", query, desc)
	}
	if name != "" {
		query = fmt.Sprintf("%s, name = '%s'", query, name)
	}

	query = fmt.Sprintf("%s RETURNING id, name, status, authorType, authorId, version, createdAt", query)
	var resp bids.BidResponse
	stmt, err = s.db.Prepare(query)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow().Scan(
		&resp.Id,
		&resp.Name,
		&resp.Status,
		&resp.AuthorType,
		&resp.AuthorId,
		&resp.Version,
		&resp.CreatedAt,
	)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

// func (s *Storage) SubmitDecision() (bids.BidResponse, error) {

// }

func (s *Storage) LeaveFeedback(bidId, bidFeedback, username string) (bids.BidResponse, error) {
	const op = "storage.postgres.LeaveFeedback"
	var uuid string
	stmt, err := s.db.Prepare(`
	SELECT id
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(username).Scan(&uuid)
	if err != nil {
		return bids.BidResponse{}, ErrUserNotFound
	}

	var bid bids.BidResponse
	var tenderId, description string
	stmt, err = s.db.Prepare(`
	SELECT id, name, status, authorType, authorId, version, createdAt, tenderId, description
	FROM bid
	WHERE id=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bidId).Scan(&bid.Id, &bid.Name, &bid.Status, &bid.AuthorType, &bid.AuthorId, &bid.Version, &bid.CreatedAt, &tenderId, &description)
	if err != nil {
		return bids.BidResponse{}, ErrNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT 1
	FROM organization_responsible o
	JOIN employee e
	ON o.user_id=e.id
	WHERE username = $1
	AND organization_id = (
		SELECT t.organizationId
		FROM tender t
		JOIN bid b ON t.id = b.tenderId
		WHERE b.id = $2
	);
  	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	var trash int
	err = stmt.QueryRow(username, bidId).Scan(&trash)
	if err != nil {
		return bids.BidResponse{}, ErrForbidden
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO feedback(bidId, description)
	VALUES ($1, $2)
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	_, err = stmt.Exec(bidId, bidFeedback)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	stmt, err = s.db.Prepare(`
	SELECT id, name, status, authorType, authorId, version, createdAt
	FROM bid
	WHERE id = $1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	var resp bids.BidResponse
	err = stmt.QueryRow(bidId).Scan(
		&resp.Id,
		&resp.Name,
		&resp.Status,
		&resp.AuthorType,
		&resp.AuthorId,
		&resp.Version,
		&resp.CreatedAt,
	)
	if err != nil {
		return bids.BidResponse{}, ErrNotFound
	}
	return resp, nil
}

func (s *Storage) RollbackBid(bidId, username string, version int) (bids.BidResponse, error) {
	const op = "storage.postgres.RollbackBid"
	var uuid string
	stmt, err := s.db.Prepare(`
	SELECT id
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(username).Scan(&uuid)
	if err != nil {
		return bids.BidResponse{}, ErrUserNotFound
	}
	stmt, err = s.db.Prepare(`
	SELECT id, name, description, status, version
	FROM bid
	WHERE id = $1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var bid, resp bids.BidResponse
	var description string
	err = stmt.QueryRow(bidId).Scan(
		&bid.Id,
		&bid.Name,
		&description,
		&bid.Status,
		&bid.Version,
	)
	if err != nil {
		return bids.BidResponse{}, ErrNotFound
	}

	stmt, err = s.db.Prepare(`
	INSERT INTO bidHistory(bidId, name, description, status, version)
	VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(bid.Id, bid.Name, description, bid.Status, bid.Version)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	SELECT  bidId, name, description, status
	FROM bidHistory
	WHERE bidId=$1 AND version=$2
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bidId, version).Scan(
		&bid.Id,
		&bid.Name,
		&description,
		&bid.Status,
	)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err = s.db.Prepare(`
	UPDATE bid
	SET version = version + 1, name = $1, description = $2, status = $3
	WHERE id = $4
	RETURNING id, name, status, authorType, authorId, version, createdAt
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bid.Name, description, bid.Status, bidId).Scan(
		&resp.Id,
		&resp.Name,
		&resp.Status,
		&resp.AuthorType,
		&resp.AuthorId,
		&resp.Version,
		&resp.CreatedAt,
	)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	return resp, nil
}

func (s *Storage) GetTenderReviews(tenderId, authorUsername, requesterUsername string, limit, offset int) ([]bids.BidReviewResponse, error) {
	const op = "storage.postgres.GetTenderReviews"
	var resp bids.BidReviewResponse
	var response []bids.BidReviewResponse

	var uuid string
	stmt, err := s.db.Prepare(`
	SELECT id
	FROM tender
	WHERE id=$1
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(tenderId).Scan(&uuid)
	if err != nil {
		return nil, ErrNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT id
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(authorUsername).Scan(&uuid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT id
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(requesterUsername).Scan(&uuid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	stmt, err = s.db.Prepare(`
	select 1 from tender t join organization_responsible o on t.organizationId = o.organization_id where o.user_id=$1 and t.id=$2
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var trash int
	err = stmt.QueryRow(uuid, tenderId).Scan(&trash)
	if err != nil {
		return nil, ErrForbidden
	}

	stmt, err = s.db.Prepare(`
	select f.id, f.description, f.createdAt from bid b 
	inner join feedback f
	on b.id = f.bidId
	inner join employee e
	on e.id=b.authorId
	where tenderId=$1 and e.username=$4
	limit $2
	offset $3;
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := stmt.Query(tenderId, limit, offset, authorUsername)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	for rows.Next() {
		err = rows.Scan(
			&resp.Id,
			&resp.Description,
			&resp.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		response = append(response, resp)
	}
	return response, nil
}

func (s *Storage) SubmitDecision(bidId, decision, username string) (bids.BidResponse, error) {
	const op = "storage.postgres.SubmitDecision"

	var bid bids.BidResponse
	var tenderId string
	stmt, err := s.db.Prepare(`
	SELECT id, name, status, authorType, authorId, version, createdAt, tenderId
	FROM bid
	WHERE id=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(bidId).Scan(
		&bid.Id,
		&bid.Name,
		&bid.Status,
		&bid.AuthorType,
		&bid.AuthorId,
		&bid.Version,
		&bid.CreatedAt,
		&tenderId,
	)
	if err != nil {
		return bids.BidResponse{}, ErrNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT id
	FROM employee
	WHERE username=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var uuid string

	err = stmt.QueryRow(username).Scan(&uuid)
	if err != nil {
		return bids.BidResponse{}, ErrUserNotFound
	}

	stmt, err = s.db.Prepare(`
	SELECT count(*)
	FROM tender t
	JOIN organization_responsible o
	ON t.organizationId = o.organization_id
	WHERE o.user_id = $1 AND t.id=$2
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var tr, count int
	err = stmt.QueryRow(uuid, tenderId).Scan(&count)
	if err != nil || count < 1 {
		return bids.BidResponse{}, ErrForbidden
	}

	stmt, err = s.db.Prepare(`
	SELECT 1
	FROM voted
	WHERE user_id=$1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	err = stmt.QueryRow(uuid).Scan(&tr)
	if err == nil {
		return bids.BidResponse{}, ErrForbidden
	}

	stmt, err = s.db.Prepare(`
	SELECT numApproved, status
	FROM decisions
	WHERE bidId = $1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	var numApproved int
	var status string
	err = stmt.QueryRow(bidId).Scan(&numApproved, &status)
	if err != nil {
		stmt, err = s.db.Prepare(`
		INSERT INTO decisions(status, bidId, numApproved)
		VALUES ($1, $2, $3)
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		st := "Pending"
		numApproved := 1
		if decision == "Rejected" {
			st = "Closed"
			numApproved = 0
		}

		_, err = stmt.Exec(st, bidId, numApproved)
		if err != nil {
			return bids.BidResponse{}, ErrBadRequest
		}

		stmt, err = s.db.Prepare(`
		INSERT INTO voted(username, user_id, decision, bidId)
		VALUES ($1, $2, $3, $4)
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}
		_, err = stmt.Exec(username, uuid, decision, bidId)
		if err != nil {
			return bids.BidResponse{}, ErrBadRequest
		}

		return bid, nil
	}

	if status == "Closed" {
		return bids.BidResponse{}, ErrForbidden
	}

	if (numApproved >= count || numApproved >= 3) && decision == "Approved" {
		stmt, err = s.db.Prepare(`
		UPDATE decisions
		SET numApproved = numApproved + 1, status = 'Closed'
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		_, err = stmt.Exec()
		if err != nil {
			return bids.BidResponse{}, ErrBadRequest
		}
		stmt, err = s.db.Prepare(`
		UPDATE tender
		SET status='Closed'
		WHERE id=$1
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		_, err = stmt.Exec(tenderId)
		if err != nil {
			return bids.BidResponse{}, ErrBadRequest
		}

		stmt, err = s.db.Prepare(`
		INSERT INTO voted(username, user_id, decision, bidId)
		VALUES ($1, $2, $3, $4)
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}
		_, err = stmt.Exec(username, uuid, decision, bidId)
		if err != nil {
			return bids.BidResponse{}, ErrBadRequest
		}

		return bid, nil
	}

	if decision == "Rejected" {
		stmt, err = s.db.Prepare(`
		UPDATE decisions
		SET status = 'Closed'
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}

		_, err = stmt.Exec()
		if err != nil {
			return bids.BidResponse{}, ErrBadRequest
		}

		stmt, err = s.db.Prepare(`
		INSERT INTO voted(username, user_id, decision, bidId)
		VALUES ($1, $2, $3, $4)
		`)
		if err != nil {
			return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
		}
		_, err = stmt.Exec(username, uuid, decision, bidId)
		if err != nil {
			return bids.BidResponse{}, ErrBadRequest
		}

		return bid, nil
	}

	stmt, err = s.db.Prepare(`
	UPDATE decisions
	SET numApproved = numApproved + 1
	`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return bids.BidResponse{}, ErrBadRequest
	}

	stmt, err = s.db.Prepare(`
		INSERT INTO voted(username, user_id, decision, bidId)
		VALUES ($1, $2, $3, $4)
		`)
	if err != nil {
		return bids.BidResponse{}, fmt.Errorf("%s: %w", op, err)
	}
	_, err = stmt.Exec(username, uuid, decision, bidId)
	if err != nil {
		return bids.BidResponse{}, ErrBadRequest
	}

	return bid, nil
}
