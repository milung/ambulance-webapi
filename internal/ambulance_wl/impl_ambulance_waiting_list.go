package ambulance_wl

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slices"
)

// CreateWaitingListEntry - Saves new entry into waiting list
func (this *implAmbulanceWaitingListAPI) CreateWaitingListEntry(ctx *gin.Context) {
	updateAmbulanceFunc(ctx, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		spanctx, span := tracer.Start(c.Request.Context(), "CreateWaitingListEntry")
		defer span.End()

		var entry WaitingListEntry

		if err := c.ShouldBindJSON(&entry); err != nil {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid request body",
				"error":   err.Error(),
			}, http.StatusBadRequest
		}

		if entry.PatientId == "" {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Patient ID is required",
			}, http.StatusBadRequest
		}

		if entry.Id == "" || entry.Id == "@new" {
			entry.Id = uuid.NewString()
		}

		if entry.WaitingSince.Before(time.Now()) {
			entry.WaitingSince = time.Now()
		}

		if entry.EstimatedDurationMinutes <= 0 {
			entry.EstimatedDurationMinutes = 15
		}

		conflictIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entry.Id == waiting.Id || entry.PatientId == waiting.PatientId
		})

		if conflictIndx >= 0 {
			return nil, gin.H{
				"status":  http.StatusConflict,
				"message": "Entry already exists",
			}, http.StatusConflict
		}

		ambulance.WaitingList = append(ambulance.WaitingList, entry)
		ambulance.reconcileWaitingList(spanctx)
		// entry was copied by value return reconciled value from the list
		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entry.Id == waiting.Id
		})
		if entryIndx < 0 {
			return nil, gin.H{
				"status":  http.StatusInternalServerError,
				"message": "Failed to save entry",
			}, http.StatusInternalServerError
		}
		return ambulance, ambulance.WaitingList[entryIndx], http.StatusOK
	})
}

// DeleteWaitingListEntry - Deletes specific entry
func (this *implAmbulanceWaitingListAPI) DeleteWaitingListEntry(ctx *gin.Context) {
	updateAmbulanceFunc(ctx, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		spanctx, span := tracer.Start(c.Request.Context(), "DeleteWaitingListEntry")
		defer span.End()

		entryId := ctx.Param("entryId")

		if entryId == "" {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Entry ID is required",
			}, http.StatusBadRequest
		}

		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entryId == waiting.Id
		})

		if entryIndx < 0 {
			return nil, gin.H{
				"status":  http.StatusNotFound,
				"message": "Entry not found",
			}, http.StatusNotFound
		}

		ambulance.WaitingList = append(ambulance.WaitingList[:entryIndx], ambulance.WaitingList[entryIndx+1:]...)
		ambulance.reconcileWaitingList(spanctx)
		return ambulance, nil, http.StatusNoContent
	})
}

// GetWaitingListEntries - Provides the ambulance waiting list
func (this *implAmbulanceWaitingListAPI) GetWaitingListEntries(ctx *gin.Context) {
	// update ambulance document
	updateAmbulanceFunc(ctx, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		_, span := tracer.Start(c.Request.Context(), "GetWaitingListEntries")
		defer span.End()

		result := ambulance.WaitingList
		if result == nil {
			result = []WaitingListEntry{}
		}
		return nil, result, http.StatusOK
	})
}

// GetWaitingListEntry - Provides details about waiting list entry
func (this *implAmbulanceWaitingListAPI) GetWaitingListEntry(ctx *gin.Context) {
	// update ambulance document
	updateAmbulanceFunc(ctx, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		_, span := tracer.Start(c.Request.Context(), "GetWaitingListEntry")
		defer span.End()

		entryId := ctx.Param("entryId")

		if entryId == "" {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Entry ID is required",
			}, http.StatusBadRequest
		}

		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entryId == waiting.Id
		})

		if entryIndx < 0 {
			return nil, gin.H{
				"status":  http.StatusNotFound,
				"message": "Entry not found",
			}, http.StatusNotFound
		}
		// return nil ambulance - no need to update it in db
		return nil, ambulance.WaitingList[entryIndx], http.StatusOK
	})
}

// UpdateWaitingListEntry - Updates specific entry
func (this *implAmbulanceWaitingListAPI) UpdateWaitingListEntry(ctx *gin.Context) {
	// update ambulance document
	updateAmbulanceFunc(ctx, func(c *gin.Context, ambulance *Ambulance) (*Ambulance, interface{}, int) {
		// special handling for gin context
		// we need to extract the span context and create a new context to ensure span context propagation
		// to the updater function
		spanctx, span := tracer.Start(
			c.Request.Context(),
			"UpdateWaitingListEntry",
			trace.WithAttributes(
				attribute.String("ambulance_id", ambulance.Id),
				attribute.String("ambulance_name", ambulance.Name),
			),
		)
		c.Request = c.Request.WithContext(spanctx)
		defer span.End()
		var entry WaitingListEntry

		if err := c.ShouldBindJSON(&entry); err != nil {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Invalid request body",
				"error":   err.Error(),
			}, http.StatusBadRequest
		}

		entryId := ctx.Param("entryId")

		if entryId == "" {
			return nil, gin.H{
				"status":  http.StatusBadRequest,
				"message": "Entry ID is required",
			}, http.StatusBadRequest
		}

		entryIndx := slices.IndexFunc(ambulance.WaitingList, func(waiting WaitingListEntry) bool {
			return entryId == waiting.Id
		})

		if entryIndx < 0 {
			return nil, gin.H{
				"status":  http.StatusNotFound,
				"message": "Entry not found",
			}, http.StatusNotFound
		}

		if entry.PatientId != "" {
			ambulance.WaitingList[entryIndx].PatientId = entry.PatientId
		}

		if entry.Id != "" {
			ambulance.WaitingList[entryIndx].Id = entry.Id
		}

		if entry.WaitingSince.After(time.Time{}) {
			ambulance.WaitingList[entryIndx].WaitingSince = entry.WaitingSince
		}

		if entry.EstimatedDurationMinutes > 0 {
			ambulance.WaitingList[entryIndx].EstimatedDurationMinutes = entry.EstimatedDurationMinutes
		}

		ambulance.reconcileWaitingList(spanctx)
		return ambulance, ambulance.WaitingList[entryIndx], http.StatusOK
	})
}
