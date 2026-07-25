package utils

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type AdminQueryParams struct {
	Pagination
	RoleID uint64 `form:"role_id"`
	Query  string `form:"search"`
}

type StateQueryParams struct {
	Pagination
	State uint64 `form:"state"`
	Query string `form:"search"`
}

type SearchQueryParams struct {
	Pagination
	Query string `form:"search"`
}

type ProjectQueryParams struct {
	Pagination
	ProjectID uint64 `form:"project_id"`
	UserID    uint64 `form:"user_id"`
	StateID   uint64 `form:"state_id"`
	CompanyID uint64 `form:"company_id"`
	Query     string `form:"search"`
}

type TaskQueryParams struct {
	Pagination
	TaskID    uint64 `form:"task_id"`
	UserID    uint64 `form:"user_id"`
	StateID   uint64 `form:"state_id"`
	ProjectID uint64 `form:"project_id"`
	TypeID    uint64 `form:"type_id"`
	DateFrom  string `form:"date_from"`
	DateTo    string `form:"date_to"`
	Query     string `form:"search"`
}

type PunchQueryParams struct {
	Pagination
	UserID   uint64 `form:"user_id"`
	FromDate string `form:"from_date"`
	ToDate   string `form:"to_date"`
}

type KeepQueryParams struct {
	Pagination
	OwnerID   uint64  `form:"owner_id"`
	ProjectID *uint64 `form:"project_id"`
	TaskID    *uint64 `form:"task_id"`
	Query     string  `form:"search"`
}

type SupplierQueryParams struct {
	Pagination
	BrandID uint64 `form:"brand_id"`
	Query   string `form:"search"`
}

func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func parseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func ParsePagination(c *gin.Context) (Pagination, error) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return Pagination{}, errors.New("parámetro 'page' inválido: debe ser un número entero positivo")
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		return Pagination{}, errors.New("parámetro 'page_size' inválido: debe ser un número entero positivo")
	}

	return Pagination{Page: page, PageSize: pageSize}, nil
}

func ParseAdminQuery(c *gin.Context) (AdminQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return AdminQueryParams{}, err
	}

	roleID, err := parseUint64(c.DefaultQuery("role_id", "0"))
	if err != nil {
		return AdminQueryParams{}, errors.New("parámetro 'role_id' inválido")
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return AdminQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return AdminQueryParams{
		Pagination: p,
		RoleID:     roleID,
		Query:      query,
	}, nil
}

func ParseStateQuery(c *gin.Context) (StateQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return StateQueryParams{}, err
	}

	state, err := parseUint64(c.DefaultQuery("state", "0"))
	if err != nil {
		return StateQueryParams{}, errors.New("parámetro 'state' inválido")
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return StateQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return StateQueryParams{
		Pagination: p,
		State:      state,
		Query:      query,
	}, nil
}

func ParseSearchQuery(c *gin.Context) (SearchQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return SearchQueryParams{}, err
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return SearchQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return SearchQueryParams{
		Pagination: p,
		Query:      query,
	}, nil
}

func ParseProjectQuery(c *gin.Context) (ProjectQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return ProjectQueryParams{}, err
	}

	projectID, err := parseUint64(c.DefaultQuery("project_id", "0"))
	if err != nil {
		return ProjectQueryParams{}, errors.New("parámetro 'project_id' inválido")
	}

	userID, err := parseUint64(c.DefaultQuery("user_id", "0"))
	if err != nil {
		return ProjectQueryParams{}, errors.New("parámetro 'user_id' inválido")
	}

	stateID, err := parseUint64(c.DefaultQuery("state_id", "0"))
	if err != nil {
		return ProjectQueryParams{}, errors.New("parámetro 'state_id' inválido")
	}

	companyID, err := parseUint64(c.DefaultQuery("company_id", "0"))
	if err != nil {
		return ProjectQueryParams{}, errors.New("parámetro 'company_id' inválido")
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return ProjectQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return ProjectQueryParams{
		Pagination: p,
		ProjectID:  projectID,
		UserID:     userID,
		StateID:    stateID,
		CompanyID:  companyID,
		Query:      query,
	}, nil
}

func ParseTaskQuery(c *gin.Context) (TaskQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return TaskQueryParams{}, err
	}

	taskID, err := parseUint64(c.DefaultQuery("task_id", "0"))
	if err != nil {
		return TaskQueryParams{}, errors.New("parámetro 'task_id' inválido")
	}

	stateID, err := parseUint64(c.DefaultQuery("state_id", "0"))
	if err != nil {
		return TaskQueryParams{}, errors.New("parámetro 'state' inválido")
	}

	projectID, err := parseUint64(c.DefaultQuery("project_id", "0"))
	if err != nil {
		return TaskQueryParams{}, errors.New("parámetro 'project_id' inválido")
	}

	userID, err := parseUint64(c.DefaultQuery("user_id", "0"))
	if err != nil {
		return TaskQueryParams{}, errors.New("parámetro 'user_id' inválido")
	}

	typeID, err := parseUint64(c.DefaultQuery("type_id", "0"))
	if err != nil {
		return TaskQueryParams{}, errors.New("parámetro 'type_id' inválido")
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return TaskQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	dateFrom := c.DefaultQuery("date_from", "")
	dateTo := c.DefaultQuery("date_to", "")

	return TaskQueryParams{
		Pagination: p,
		TaskID:     taskID,
		ProjectID:  projectID,
		UserID:     userID,
		StateID:    stateID,
		TypeID:     typeID,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Query:      query,
	}, nil
}

func ParsePunchQuery(c *gin.Context) (PunchQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return PunchQueryParams{}, err
	}

	userID, err := parseUint64(c.DefaultQuery("user_id", "0"))
	if err != nil {
		return PunchQueryParams{}, errors.New("parámetro 'user_id' inválido")
	}

	fromDate := c.DefaultQuery("from_date", "")
	toDate := c.DefaultQuery("to_date", "")

	return PunchQueryParams{
		Pagination: p,
		UserID:     userID,
		FromDate:   fromDate,
		ToDate:     toDate,
	}, nil
}

func ParseKeepQuery(c *gin.Context) (KeepQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return KeepQueryParams{}, err
	}

	ownerID, err := parseUint64(c.DefaultQuery("owner_id", "0"))
	if err != nil {
		return KeepQueryParams{}, errors.New("parámetro 'owner_id' inválido")
	}

	var projectID *uint64
	if projectIDStr := c.DefaultQuery("project_id", ""); projectIDStr != "" {
		projectIDVal, err := parseUint64(projectIDStr)
		if err != nil {
			return KeepQueryParams{}, errors.New("parámetro 'project_id' inválido")
		}
		projectID = &projectIDVal
	}

	var taskID *uint64
	if taskIDStr := c.DefaultQuery("task_id", ""); taskIDStr != "" {
		taskIDVal, err := parseUint64(taskIDStr)
		if err != nil {
			return KeepQueryParams{}, errors.New("parámetro 'task_id' inválido")
		}
		taskID = &taskIDVal
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return KeepQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return KeepQueryParams{
		Pagination: p,
		OwnerID:    ownerID,
		ProjectID:  projectID,
		TaskID:     taskID,
		Query:      query,
	}, nil
}

func ParseSupplierQuery(c *gin.Context) (SupplierQueryParams, error) {
	p, err := ParsePagination(c)
	if err != nil {
		return SupplierQueryParams{}, err
	}

	brandID, err := parseUint64(c.DefaultQuery("brand_id", "0"))
	if err != nil {
		return SupplierQueryParams{}, errors.New("parámetro 'brand_id' inválido")
	}

	query, err := NormalizeSearch(c.DefaultQuery("search", ""))
	if err != nil {
		return SupplierQueryParams{}, fmt.Errorf("parámetro 'search' inválido: %v", err)
	}

	return SupplierQueryParams{
		Pagination: p,
		BrandID:    brandID,
		Query:      query,
	}, nil
}
