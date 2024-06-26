package lib

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"movinglake.com/haven/handler/jsonutils"
	"movinglake.com/haven/wrappers"
)

// Has the actual logic of the API in easy to test functions.

// We need to hold DB connections.
type HavenHandler struct {
	db wrappers.DB
}

func NewHavenHandler(db wrappers.DB) *HavenHandler {
	return &HavenHandler{
		db: db,
	}
}

// APIRouter is the main entry point for the API.
// We use RPCs to interact with the API. So all requests are POSTs and errors are returned as JSON.
// We always return a status 200.
func (h *HavenHandler) APIRouter(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}
	if v, ok := payload["action"]; !ok {
		c.JSON(http.StatusOK, gin.H{"error": "no action specified"})
		return
	} else {
		switch v {
		case "addPayload":
			if err := h.addPayload(payload); err != nil {
				c.JSON(http.StatusOK, gin.H{"error": err.Error()})
				return
			}
		case "validatePayload":
			if err := h.addPayload(payload); err != nil {
				c.JSON(http.StatusOK, gin.H{"error": err.Error()})
				return
			}
		default:
			c.JSON(http.StatusOK, gin.H{"error": "unknown action " + v.(string)})
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// addPayload adds a new payload to the specific resource.
func (h *HavenHandler) addPayload(p map[string]interface{}) error {
	resource := p["resource"].(string)
	payload := p["payload"]

	// Get the schema of the resource.
	t := h.db.OpenTxn()
	r := &wrappers.Resource{}
	t.Find(r, "name = ?", resource)
	if r.ID == 0 {
		return fmt.Errorf("resource not found for %s", resource)
	}

	schema := make(map[string]interface{})
	if err := json.Unmarshal([]byte(r.Schema), &schema); err != nil {
		return err
	}

	newSchema, err := jsonutils.ApplyPayload(schema, payload)
	if err != nil {
		return err
	}

	if newSchema == nil {
		// No changes to the schema.
		return nil
	}

	// Save the new schema.
	newSchemaBytes, err := json.Marshal(newSchema)
	if err != nil {
		return err
	}
	r.Version += 1
	oldSchema := r.Schema
	r.Schema = string(newSchemaBytes)
	t.Save(r)

	// Save the reference payload.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	refPayload := &wrappers.ReferencePayloads{
		ResourceID: r.ID,
		Payload:    string(payloadBytes),
	}
	t.Save(refPayload)

	// Save the new version.
	rv := &wrappers.ResourceVersions{
		ResourceID:          r.ID,
		ReferencePayloadsID: refPayload.ID,
		OldSchema:           oldSchema,
		NewSchema:           string(newSchemaBytes),
		Version:             r.Version,
	}
	t.Save(rv)
	t.Commit()
	return nil
}

// validatePayload validates the payload against the schema.
func (h *HavenHandler) validatePayload(p map[string]any) error {
	resource := p["resource"].(string)
	payload := p["payload"]

	// Get the schema of the resource.
	res, err := h.db.GetResource(resource)
	if err != nil {
		return fmt.Errorf("failed to get resource from db: %w", err)
	}

	schema := make(map[string]any)
	if err := json.Unmarshal([]byte(res.Schema), &schema); err != nil {
		return err
	}

	result, err := jsonutils.ValidatePayload(schema, payload)
	if err != nil {
		return err
	}
	if !result.Valid() {
		return fmt.Errorf("payload is invalid: %v", result.Errors())
	}

	return nil
}

func (h *HavenHandler) RegisterRoutes(e *gin.Engine) error {
	e.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OK",
		})
	})
	e.POST("/api/v1", h.APIRouter)
	return nil
}
