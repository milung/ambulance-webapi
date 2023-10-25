package ambulance_wl

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetConditions - Provides the list of conditions associated with ambulance
func (this *implAmbulanceConditionsAPI) GetConditions(ctx *gin.Context) {
	//ctx.AbortWithStatus(http.StatusNotImplemented)

	updateAmbulanceFunc(ctx, func(
		ctx *gin.Context,
		ambulance *Ambulance,
	) (updatedAmbulance *Ambulance, responseContent interface{}, status int) {
		result := ambulance.PredefinedConditions
		if result == nil {
			result = []Condition{}
		}
		return nil, result, http.StatusOK
	})

}
