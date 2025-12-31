package settings

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
			case "email":
				message = field + " must be a valid email address"
			case "min":
				message = field + " must be at least " + e.Param() + " characters"
			case "max":
				message = field + " must be at most " + e.Param() + " characters"
			case "oneof":
				// Special handling for type field
				if field == "Type" {
					message = "type must be one of: string, number, boolean, json"
				} else {
					validValues := strings.ReplaceAll(e.Param(), " ", ", ")
					message = field + " must be one of: " + validValues
				}
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
		// Generic error
		errors = append(errors, response.ResponseError{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
		})
	}

	return errors
}

// @Summary Create system setting
// @Description Create a new system-wide setting (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateSettingRequest true "Setting details"
// @Success 201 {object} response.Response{data=object{setting=SettingResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/system [post]
func (m *SettingsModule) createSystemSetting(c *gin.Context) {
	var req CreateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	setting, err := m.service.CreateSystemSetting(&req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "System setting created successfully", gin.H{
		"setting": setting,
	})
}

// @Summary Get system setting
// @Description Get a specific system setting by key (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Param key path string true "Setting key"
// @Success 200 {object} response.Response{data=object{setting=SettingResponse}}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/system/{key} [get]
func (m *SettingsModule) getSystemSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Setting key is required")
		return
	}

	setting, err := m.service.GetSystemSetting(key)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "System setting retrieved successfully", gin.H{
		"setting": setting,
	})
}

// @Summary List system settings
// @Description Get all system settings with pagination (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=SettingsListResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/system [get]
func (m *SettingsModule) listSystemSettings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	settings, err := m.service.ListSystemSettings(page, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "System settings retrieved successfully", settings)
}

// @Summary Update system setting
// @Description Update an existing system setting by key (admin only)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "Setting key"
// @Param request body UpdateSettingRequest true "Updated setting details"
// @Success 200 {object} response.Response{data=object{setting=SettingResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/system/{key} [put]
func (m *SettingsModule) updateSystemSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Setting key is required")
		return
	}

	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	setting, err := m.service.UpdateSystemSetting(key, &req)
	if err != nil {
		if err.Error() == "system setting not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "System setting updated successfully", gin.H{
		"setting": setting,
	})
}

// @Summary Delete system setting
// @Description Delete a system setting by key (admin only)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Param key path string true "Setting key"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/system/{key} [delete]
func (m *SettingsModule) deleteSystemSetting(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Setting key is required")
		return
	}

	err := m.service.DeleteSystemSetting(key)
	if err != nil {
		if err.Error() == "system setting not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "System setting deleted successfully", nil)
}

// @Summary Get user setting
// @Description Get a specific user setting by key (authenticated users can only access their own settings)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Param key path string true "Setting key"
// @Success 200 {object} response.Response{data=object{setting=SettingResponse}}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/user/{key} [get]
func (m *SettingsModule) getUserSetting(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Setting key is required")
		return
	}

	setting, err := m.service.GetUserSetting(userID.(string), key)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User setting retrieved successfully", gin.H{
		"setting": setting,
	})
}

// @Summary List user settings
// @Description Get all settings for the authenticated user
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.Response{data=SettingsListResponse}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/user [get]
func (m *SettingsModule) listUserSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	settings, err := m.service.ListUserSettings(userID.(string), page, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User settings retrieved successfully", settings)
}

// @Summary Create or update user setting
// @Description Create or update a user setting by key (authenticated users can only modify their own settings)
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "Setting key"
// @Param request body UpdateSettingRequest true "Setting details"
// @Success 200 {object} response.Response{data=object{setting=SettingResponse}}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/user/{key} [put]
func (m *SettingsModule) createOrUpdateUserSetting(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Setting key is required")
		return
	}

	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, getValidationErrors(err))
		return
	}

	setting, err := m.service.CreateOrUpdateUserSetting(userID.(string), key, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "User setting saved successfully", gin.H{
		"setting": setting,
	})
}

// @Summary Delete user setting
// @Description Delete a user setting by key (authenticated users can only delete their own settings)
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Param key path string true "Setting key"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /settings/user/{key} [delete]
func (m *SettingsModule) deleteUserSetting(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	key := c.Param("key")
	if key == "" {
		response.BadRequest(c, "Setting key is required")
		return
	}

	err := m.service.DeleteUserSetting(userID.(string), key)
	if err != nil {
		if err.Error() == "user setting not found" {
			response.NotFound(c, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, "User setting deleted successfully", nil)
}
