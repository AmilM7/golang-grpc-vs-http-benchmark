package httptransport

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"golang-grpc/internal/service"
	"golang-grpc/internal/user"
)

func NewRouter(svc *service.Service) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	handler := &handler{svc: svc}

	router.GET("/healthz", handler.health)
	router.POST("/users", handler.createUser)
	router.GET("/users", handler.listUsers)
	router.GET("/users/:id", handler.getUser)
	router.PUT("/users/:id", handler.updateUser)
	router.DELETE("/users/:id", handler.deleteUser)

	return router
}

type handler struct {
	svc *service.Service
}

func (h *handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *handler) createUser(c *gin.Context) {
	var payload user.Attributes
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	created, err := h.svc.Create(c.Request.Context(), payload)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *handler) listUsers(c *gin.Context) {
	users := h.svc.List(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *handler) getUser(c *gin.Context) {
	id := c.Param("id")
	u, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *handler) updateUser(c *gin.Context) {
	id := c.Param("id")
	var payload user.Attributes
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	updated, err := h.svc.Update(c.Request.Context(), id, payload)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *handler) deleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, user.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
