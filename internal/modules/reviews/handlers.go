package reviews

import (
	"net/http"
	"strconv"

	"gogin/internal/response"

	"github.com/gin-gonic/gin"
)

// @Summary Create Review
// @Tags Reviews
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateReviewRequest true "Review details"
// @Success 201 {object} response.Response{data=ReviewResponse}
// @Router /reviews [post]
func (m *ReviewsModule) createReview(c *gin.Context) {
	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{response.NewError("VALIDATION_ERROR", err.Error(), "")})
		return
	}
	userID, _ := c.Get("user_id")
	review, err := m.service.CreateReview(userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Review created successfully", review)
}

// @Summary List Reviews
// @Tags Reviews
// @Produce json
// @Param resource_type query string false "Resource type"
// @Param resource_id query string false "Resource ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} response.Response{data=ReviewsListResponse}
// @Router /reviews [get]
func (m *ReviewsModule) listReviews(c *gin.Context) {
	resourceType := c.Query("resource_type")
	resourceID := c.Query("resource_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Validate that both resource_type and resource_id are provided together
	if (resourceType != "" && resourceID == "") || (resourceType == "" && resourceID != "") {
		response.BadRequest(c, "Both resource_type and resource_id must be provided together")
		return
	}

	reviews, total, avgRating, err := m.service.ListReviews(resourceType, resourceID, page, limit)
	if err != nil {
		response.InternalError(c, "Failed to list reviews")
		return
	}

	response.Success(c, http.StatusOK, "Reviews retrieved", gin.H{
		"reviews":        reviews,
		"total":          total,
		"average_rating": avgRating,
		"page":           page,
		"limit":          limit,
		"total_pages":    (total + limit - 1) / limit,
	})
}

// @Summary Get Review
// @Tags Reviews
// @Produce json
// @Param id path string true "Review ID"
// @Success 200 {object} response.Response{data=ReviewResponse}
// @Router /reviews/{id} [get]
func (m *ReviewsModule) getReview(c *gin.Context) {
	review, err := m.service.GetReview(c.Param("id"))
	if err != nil {
		response.NotFound(c, "Review not found")
		return
	}
	response.Success(c, http.StatusOK, "Review retrieved", review)
}

// @Summary Update Review
// @Tags Reviews
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Review ID"
// @Param request body UpdateReviewRequest true "Review update"
// @Success 200 {object} response.Response{data=ReviewResponse}
// @Router /reviews/{id} [put]
func (m *ReviewsModule) updateReview(c *gin.Context) {
	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, []response.ResponseError{response.NewError("VALIDATION_ERROR", err.Error(), "")})
		return
	}
	userID, _ := c.Get("user_id")
	review, err := m.service.UpdateReview(c.Param("id"), userID.(string), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Review updated", review)
}

// @Summary Delete Review
// @Tags Reviews
// @Produce json
// @Security BearerAuth
// @Param id path string true "Review ID"
// @Success 200 {object} response.Response
// @Router /reviews/{id} [delete]
func (m *ReviewsModule) deleteReview(c *gin.Context) {
	userID, _ := c.Get("user_id")
	if err := m.service.DeleteReview(c.Param("id"), userID.(string)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Review deleted", nil)
}
