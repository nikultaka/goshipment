package tickets

import (
	"net/http"
	"strconv"
	"strings"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// getValidationErrors extracts detailed validation error messages
func getValidationErrors(err error) []response.ResponseError {
	var errors []response.ResponseError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			var message string
			field := e.Field()

			switch e.Tag() {
			case "required":
				message = field + " is required"
			case "min":
				message = field + " must be at least " + e.Param() + " characters"
			case "max":
				message = field + " must be at most " + e.Param() + " characters"
			case "oneof":
				if field == "Priority" {
					message = "priority must be one of: low, medium, high, urgent"
				} else if field == "Status" {
					message = "status must be one of: open, in_progress, resolved, closed"
				} else {
					validValues := strings.ReplaceAll(e.Param(), " ", ", ")
					message = field + " must be one of: " + validValues
				}
			case "uuid":
				message = field + " must be a valid UUID"
			default:
				message = field + " is invalid"
			}

			errors = append(errors, response.ResponseError{
				Code:    "VALIDATION_ERROR",
				Message: message,
				Field:   strings.ToLower(field),
			})
		}
	} else {
		errors = append(errors, response.ResponseError{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}

	return errors
}

// @Summary Create support ticket
// @Description Create a new support ticket
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTicketRequest true "Ticket details"
// @Success 201 {object} response.Response{data=object{ticket=TicketResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets [post]
func (m *TicketsModule) createTicket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	ticket, err := m.service.CreateTicket(userID.(string), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Ticket created successfully", gin.H{
		"ticket": ticket,
	})
}

// @Summary Get ticket details
// @Description Get a specific ticket with all replies
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Success 200 {object} response.Response{data=TicketDetailResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id} [get]
func (m *TicketsModule) getTicket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	role, _ := c.Get("role")
	ticketID := c.Param("id")

	// Get ticket with replies
	ticketDetail, err := m.service.GetTicketWithReplies(ticketID)
	if err != nil {
		if err.Error() == "ticket not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	// Check authorization: user can only view their own tickets unless admin
	if role != "admin" && ticketDetail.Ticket.UserID != userID.(string) {
		response.Forbidden(c, "Access denied")
		return
	}

	response.Success(c, http.StatusOK, "Ticket retrieved successfully", ticketDetail)
}

// @Summary List my tickets
// @Description List all tickets created by the authenticated user
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(open, in_progress, resolved, closed)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=TicketsListResponse}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/my [get]
func (m *TicketsModule) listMyTickets(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	tickets, err := m.service.ListUserTickets(userID.(string), status, page, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Tickets retrieved successfully", tickets)
}

// @Summary List all tickets
// @Description List all support tickets (admin only)
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(open, in_progress, resolved, closed)
// @Param priority query string false "Filter by priority" Enums(low, medium, high, urgent)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=TicketsListResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets [get]
func (m *TicketsModule) listAllTickets(c *gin.Context) {
	status := c.Query("status")
	priority := c.Query("priority")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	tickets, err := m.service.ListAllTickets(status, priority, page, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Tickets retrieved successfully", tickets)
}

// @Summary Update ticket
// @Description Update ticket details (users can only update their own open tickets)
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param request body UpdateTicketRequest true "Updated ticket details"
// @Success 200 {object} response.Response{data=object{ticket=TicketResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id} [put]
func (m *TicketsModule) updateTicket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ticketID := c.Param("id")

	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	ticket, err := m.service.UpdateTicket(ticketID, userID.(string), &req)
	if err != nil {
		if err.Error() == "ticket not found or access denied" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Ticket updated successfully", gin.H{
		"ticket": ticket,
	})
}

// @Summary Update ticket status
// @Description Update the status of a ticket (admin only)
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param request body UpdateTicketStatusRequest true "Status update"
// @Success 200 {object} response.Response{data=object{ticket=TicketResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id}/status [put]
func (m *TicketsModule) updateTicketStatus(c *gin.Context) {
	ticketID := c.Param("id")

	var req UpdateTicketStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	ticket, err := m.service.UpdateTicketStatus(ticketID, &req)
	if err != nil {
		if err.Error() == "ticket not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Ticket status updated successfully", gin.H{
		"ticket": ticket,
	})
}

// @Summary Assign ticket
// @Description Assign a ticket to an admin (admin only)
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param request body AssignTicketRequest true "Assignment details"
// @Success 200 {object} response.Response{data=object{ticket=TicketResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id}/assign [put]
func (m *TicketsModule) assignTicket(c *gin.Context) {
	ticketID := c.Param("id")

	var req AssignTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	ticket, err := m.service.AssignTicket(ticketID, &req)
	if err != nil {
		if err.Error() == "ticket not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Ticket assigned successfully", gin.H{
		"ticket": ticket,
	})
}

// @Summary Add reply to ticket
// @Description Add a reply to a support ticket
// @Tags Tickets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param request body CreateReplyRequest true "Reply content"
// @Success 201 {object} response.Response{data=object{reply=ReplyResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id}/replies [post]
func (m *TicketsModule) createReply(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	role, _ := c.Get("role")
	ticketID := c.Param("id")

	var req CreateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	// Check if user has access to this ticket
	ticket, err := m.service.GetTicketByID(ticketID)
	if err != nil {
		if err.Error() == "ticket not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	// Only ticket owner or admin can reply
	if role != "admin" && ticket.UserID != userID.(string) {
		response.Forbidden(c, "Access denied")
		return
	}

	// Determine if reply is from staff
	isStaff := role == "admin"

	reply, err := m.service.CreateReply(ticketID, userID.(string), isStaff, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Reply added successfully", gin.H{
		"reply": reply,
	})
}

// @Summary Delete ticket
// @Description Delete a ticket (users can only delete their own open tickets)
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id} [delete]
func (m *TicketsModule) deleteTicket(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ticketID := c.Param("id")

	err := m.service.DeleteTicket(ticketID, userID.(string))
	if err != nil {
		if err.Error() == "ticket not found or cannot be deleted" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Ticket deleted successfully", nil)
}

// @Summary Upload attachment
// @Description Upload a file attachment to a support ticket
// @Tags Tickets
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param file formData file true "File to upload"
// @Param attachment_type formData string false "Attachment type" Enums(document, screenshot, log_file) default(document)
// @Param description formData string false "Description of the attachment"
// @Success 201 {object} response.Response{data=object{attachment=AttachmentResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id}/attachments [post]
func (m *TicketsModule) uploadAttachment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ticketID := c.Param("id")

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File is required")
		return
	}

	// Get optional fields
	attachmentType := c.DefaultPostForm("attachment_type", "document")
	description := c.PostForm("description")

	// Validate attachment type
	validTypes := map[string]bool{"document": true, "screenshot": true, "log_file": true}
	if !validTypes[attachmentType] {
		response.BadRequest(c, "attachment_type must be one of: document, screenshot, log_file")
		return
	}

	attachment, err := m.service.UploadAttachment(ticketID, userID.(string), file, attachmentType, description)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "not found") {
			response.Forbidden(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "Attachment uploaded successfully", gin.H{
		"attachment": attachment,
	})
}

// @Summary List attachments
// @Description List all attachments for a support ticket
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Success 200 {object} response.Response{data=AttachmentsListResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id}/attachments [get]
func (m *TicketsModule) listAttachments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ticketID := c.Param("id")

	attachments, err := m.service.ListAttachments(ticketID, userID.(string))
	if err != nil {
		if strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "not found") {
			response.Forbidden(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Attachments retrieved successfully", attachments)
}

// @Summary Delete attachment
// @Description Delete an attachment from a support ticket
// @Tags Tickets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Ticket ID"
// @Param attachment_id path string true "Attachment ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /tickets/{id}/attachments/{attachment_id} [delete]
func (m *TicketsModule) deleteAttachment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ticketID := c.Param("id")
	attachmentID := c.Param("attachment_id")

	err := m.service.DeleteAttachment(ticketID, attachmentID, userID.(string))
	if err != nil {
		if strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "not found") {
			response.Forbidden(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "Attachment deleted successfully", nil)
}

// ========== Admin Ticket Endpoints ==========

// @Summary Create ticket and assign (Admin)
// @Description Admin creates a new support ticket and optionally assigns it
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AdminCreateTicketRequest true "Ticket details"
// @Success 201 {object} response.Response{data=object{ticket=TicketResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/tickets [post]
func (m *TicketsModule) adminCreateTicket(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Admin not authenticated")
		return
	}

	var req AdminCreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	ticket, err := m.service.AdminCreateTicket(adminID.(string), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Ticket created and assigned successfully", gin.H{
		"ticket": ticket,
	})
}

// @Summary Admin reply/message (Admin)
// @Description Admin adds a reply or message to a ticket
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ticket_id path string true "Ticket ID"
// @Param request body CreateReplyRequest true "Message content"
// @Success 201 {object} response.Response{data=object{reply=ReplyResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/tickets/{ticket_id}/messages [post]
func (m *TicketsModule) adminCreateMessage(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Admin not authenticated")
		return
	}

	ticketID := c.Param("ticket_id")

	var req CreateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	reply, err := m.service.AdminCreateReply(ticketID, adminID.(string), req.Content)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "Message added successfully", gin.H{
		"reply": reply,
	})
}

// @Summary Attach file (Admin)
// @Description Admin attaches a file to a ticket
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param ticket_id path string true "Ticket ID"
// @Param file formData file true "File to upload"
// @Param attachment_type formData string false "Attachment type" Enums(document, screenshot, log_file) default(document)
// @Param description formData string false "Description of the attachment"
// @Success 201 {object} response.Response{data=object{attachment=AttachmentResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/tickets/{ticket_id}/attachments [post]
func (m *TicketsModule) adminUploadAttachment(c *gin.Context) {
	adminID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Admin not authenticated")
		return
	}

	ticketID := c.Param("ticket_id")

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "File is required")
		return
	}

	// Get optional fields
	attachmentType := c.DefaultPostForm("attachment_type", "document")
	description := c.PostForm("description")

	// Validate attachment type
	validTypes := map[string]bool{"document": true, "screenshot": true, "log_file": true}
	if !validTypes[attachmentType] {
		response.BadRequest(c, "attachment_type must be one of: document, screenshot, log_file")
		return
	}

	attachment, err := m.service.AdminUploadAttachment(ticketID, adminID.(string), file, attachmentType, description)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, "Attachment uploaded successfully", gin.H{
		"attachment": attachment,
	})
}

// @Summary List tickets for specific user (Admin)
// @Description Get all tickets created by a specific user
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param status query string false "Filter by status"
// @Param priority query string false "Filter by priority"
// @Success 200 {object} response.Response{data=TicketsListResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/users/{user_id}/tickets [get]
func (m *TicketsModule) adminGetUserTickets(c *gin.Context) {
	userID := c.Param("id")

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Parse filters
	status := c.Query("status")
	priority := c.Query("priority")

	tickets, total, err := m.service.GetTicketsByUser(userID, page, limit, status, priority)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	totalPages := (total + limit - 1) / limit

	response.Success(c, http.StatusOK, "User tickets retrieved successfully", TicketsListResponse{
		Tickets:    tickets,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}
