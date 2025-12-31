package apiclient

import (
	"net/http"
	"strconv"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// createClient creates a new OAuth client
// @Summary Create API Client
// @Description Create a new OAuth 2.0 client application (admin only)
// @Tags API Clients
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateClientRequest true "Client details"
// @Success 201 {object} response.Response{data=ClientResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /clients [post]
func (m *APIClientModule) createClient(c *gin.Context) {
	var req CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	userID, _ := c.Get("user_id")
	client, err := m.service.CreateClient(userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Client created successfully", client)
}

// listClients lists all OAuth clients
// @Summary List API Clients
// @Description Get a paginated list of all OAuth clients (admin only)
// @Tags API Clients
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=ClientsListResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /clients [get]
func (m *APIClientModule) listClients(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	clients, total, err := m.service.ListClients(page, limit)
	if err != nil {
		response.InternalError(c, "Failed to list clients")
		return
	}

	totalPages := (total + limit - 1) / limit

	response.Success(c, http.StatusOK, "Clients retrieved successfully", gin.H{
		"clients":     clients,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

// getClient retrieves a client by ID
// @Summary Get API Client
// @Description Get an OAuth client by ID (admin only)
// @Tags API Clients
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Success 200 {object} response.Response{data=ClientResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /clients/{id} [get]
func (m *APIClientModule) getClient(c *gin.Context) {
	id := c.Param("id")

	client, err := m.service.GetClient(id)
	if err != nil {
		response.NotFound(c, "Client not found")
		return
	}

	response.Success(c, http.StatusOK, "Client retrieved successfully", client)
}

// updateClient updates a client
// @Summary Update API Client
// @Description Update an OAuth client (admin only)
// @Tags API Clients
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Param request body UpdateClientRequest true "Client update details"
// @Success 200 {object} response.Response{data=ClientResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /clients/{id} [put]
func (m *APIClientModule) updateClient(c *gin.Context) {
	id := c.Param("id")

	var req UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	client, err := m.service.UpdateClient(id, &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Client updated successfully", client)
}

// deleteClient deletes a client
// @Summary Delete API Client
// @Description Delete an OAuth client (admin only)
// @Tags API Clients
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /clients/{id} [delete]
func (m *APIClientModule) deleteClient(c *gin.Context) {
	id := c.Param("id")

	err := m.service.DeleteClient(id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Client deleted successfully", nil)
}

// regenerateSecret regenerates client secret
// @Summary Regenerate Client Secret
// @Description Generate a new secret for an OAuth client (admin only)
// @Tags API Clients
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Success 200 {object} response.Response{data=object{client_secret=string}}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /clients/{id}/regenerate-secret [post]
func (m *APIClientModule) regenerateSecret(c *gin.Context) {
	id := c.Param("id")

	newSecret, err := m.service.RegenerateSecret(id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Client secret regenerated successfully", gin.H{
		"client_secret": newSecret,
	})
}

// updateStatus updates client status
// @Summary Update Client Status
// @Description Activate or deactivate an OAuth client (admin only)
// @Tags API Clients
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Client ID"
// @Param request body object{is_active=bool} true "Status update"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /clients/{id}/status [put]
func (m *APIClientModule) updateStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		IsActive bool `json:"is_active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.UpdateStatus(id, req.IsActive)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Client status updated successfully", gin.H{
		"is_active": req.IsActive,
	})
}
