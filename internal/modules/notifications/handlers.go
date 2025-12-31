package notifications

import (
	"net/http"
	"strconv"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// createNotification creates a new notification
// @Summary Create Notification
// @Description Create and send a new notification
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body SendNotificationRequest true "Notification details"
// @Success 201 {object} response.Response{data=NotificationResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /notifications [post]
func (m *NotificationsModule) createNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	notif, err := m.service.SendNotification(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Notification created successfully", notif)
}

// listNotifications lists user notifications
// @Summary List Notifications
// @Description Get paginated list of user notifications
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=NotificationsListResponse}
// @Failure 401 {object} response.Response
// @Router /notifications [get]
func (m *NotificationsModule) listNotifications(c *gin.Context) {
	userID, _ := c.Get("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	notifications, total, unread, err := m.service.ListNotifications(userID.(string), page, limit)
	if err != nil {
		response.InternalError(c, "Failed to list notifications")
		return
	}

	totalPages := (total + limit - 1) / limit

	response.Success(c, http.StatusOK, "Notifications retrieved successfully", gin.H{
		"notifications": notifications,
		"total":         total,
		"unread":        unread,
		"page":          page,
		"limit":         limit,
		"total_pages":   totalPages,
	})
}

// getNotification retrieves a notification by ID
// @Summary Get Notification
// @Description Get a notification by ID
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} response.Response{data=NotificationResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /notifications/{id} [get]
func (m *NotificationsModule) getNotification(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	notif, err := m.service.GetNotification(id, userID.(string))
	if err != nil {
		response.NotFound(c, "Notification not found")
		return
	}

	response.Success(c, http.StatusOK, "Notification retrieved successfully", notif)
}

// markAsRead marks a notification as read
// @Summary Mark Notification as Read
// @Description Mark a notification as read
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /notifications/{id}/read [put]
func (m *NotificationsModule) markAsRead(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	err := m.service.MarkAsRead(id, userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Notification marked as read", nil)
}

// markAllAsRead marks all notifications as read for the authenticated user
// @Summary Mark all notifications as read
// @Description Mark all unread notifications as read for the authenticated user
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=object{marked_count=int}}
// @Failure 401 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /notifications/mark-all-read [put]
func (m *NotificationsModule) markAllAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	markedCount, err := m.service.MarkAllAsRead(userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "All notifications marked as read", gin.H{
		"marked_count": markedCount,
	})
}

// deleteNotification deletes a notification
// @Summary Delete Notification
// @Description Delete a notification
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /notifications/{id} [delete]
func (m *NotificationsModule) deleteNotification(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	err := m.service.DeleteNotification(id, userID.(string))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Notification deleted successfully", nil)
}

// testEmail sends a test email
// @Summary Test Email
// @Description Send a test email via SendGrid
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body TestEmailRequest true "Email details"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /notifications/test-email [post]
func (m *NotificationsModule) testEmail(c *gin.Context) {
	var req TestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.SendEmail([]string{req.To}, req.Subject, req.Body)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Test email sent successfully", nil)
}

// testSMS sends a test SMS
// @Summary Test SMS
// @Description Send a test SMS via Twilio
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body TestSMSRequest true "SMS details"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /notifications/test-sms [post]
func (m *NotificationsModule) testSMS(c *gin.Context) {
	var req TestSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	err := m.service.SendSMS(req.To, req.Body)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Test SMS sent successfully", nil)
}

// Admin Notification Template Handlers

// listNotificationTemplates lists all notification templates
// @Summary List notification templates
// @Description Get paginated list of notification templates (Admin only)
// @Tags Admin - Notification Templates
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Success 200 {object} response.Response{data=NotificationTemplatesListResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/notification-templates [get]
func (m *NotificationsModule) listNotificationTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	templates, err := m.service.ListNotificationTemplates(page, limit)
	if err != nil {
		response.InternalError(c, "Failed to list notification templates: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Notification templates retrieved successfully", templates)
}

// createNotificationTemplate creates a new notification template
// @Summary Create notification template
// @Description Create a new notification template (Admin only)
// @Tags Admin - Notification Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateNotificationTemplateRequest true "Template details"
// @Success 201 {object} response.Response{data=NotificationTemplateResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /admin/notification-templates [post]
func (m *NotificationsModule) createNotificationTemplate(c *gin.Context) {
	var req CreateNotificationTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	template, err := m.service.CreateNotificationTemplate(&req, userID.(string))
	if err != nil {
		response.BadRequest(c, "Failed to create notification template: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Notification template created successfully", template)
}

// updateNotificationTemplate updates an existing notification template
// @Summary Update notification template
// @Description Update an existing notification template (Admin only)
// @Tags Admin - Notification Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Param request body UpdateNotificationTemplateRequest true "Template update details"
// @Success 200 {object} response.Response{data=NotificationTemplateResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response{errors=[]response.ResponseError}
// @Router /admin/notification-templates/{id} [put]
func (m *NotificationsModule) updateNotificationTemplate(c *gin.Context) {
	id := c.Param("id")

	var req UpdateNotificationTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors := []response.ResponseError{
			response.NewError("VALIDATION_ERROR", err.Error(), ""),
		}
		response.ValidationError(c, errors)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	template, err := m.service.UpdateNotificationTemplate(id, &req, userID.(string))
	if err != nil {
		response.BadRequest(c, "Failed to update notification template: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Notification template updated successfully", template)
}
