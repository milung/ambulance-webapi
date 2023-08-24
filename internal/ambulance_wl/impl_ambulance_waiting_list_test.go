package ambulance_wl

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AmbulanceWlSuite struct {
	suite.Suite
	dbServiceMock *DbServiceMock
}

func TestAmbulanceWlSuite(t *testing.T) {
	suite.Run(t, new(AmbulanceWlSuite))
}

type DbServiceMock struct {
	mock.Mock
}

func (this *DbServiceMock) CreateDocument(ctx context.Context, id string, document interface{}) error {
	args := this.Called(ctx, id, document)
	return args.Error(0)
}

func (this *DbServiceMock) FindDocument(ctx context.Context, id string) (interface{}, error) {
	args := this.Called(ctx, id)
	return args.Get(0), args.Error(1)
}

func (this *DbServiceMock) UpdateDocument(ctx context.Context, id string, document interface{}) error {
	args := this.Called(ctx, id, document)
	return args.Error(0)
}

func (this *DbServiceMock) DeleteDocument(ctx context.Context, id string) error {
	args := this.Called(ctx, id)
	return args.Error(0)
}

func (suite *AmbulanceWlSuite) SetupTest() {
	suite.dbServiceMock = &DbServiceMock{}

	suite.dbServiceMock.
		On("FindDocument", mock.Anything, mock.Anything).
		Return(
			&Ambulance{
				Id: "test-ambulance",
				WaitingList: []WaitingListEntry{
					{
						Id:                       "test-entry",
						PatientId:                "test-patient",
						WaitingSince:             time.Now(),
						EstimatedDurationMinutes: 101,
					},
				},
			},
			nil,
		)
}
func (suite *AmbulanceWlSuite) Test_UpdateWl_DbServiceUpdateCalled() {
	// ARRANGE
	suite.dbServiceMock.
		On("UpdateDocument", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	json := `{
		"id": "test-entry",
		"patientId": "test-patient",
		"estimatedDurationMinutes": 42
	}`

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("db_service", suite.dbServiceMock)
	ctx.Params = []gin.Param{
		{Key: "ambulanceId", Value: "test-ambulance"},
		{Key: "entryId", Value: "test-entry"},
	}
	ctx.Request = httptest.NewRequest("POST", "/ambulance/test-ambulance/waitinglist/test-entry", strings.NewReader(json))

	sut := implAmbulanceWaitingListAPI{}

	// ACT
	sut.UpdateWaitingListEntry(ctx)

	// ASSERT
	suite.dbServiceMock.AssertCalled(suite.T(), "UpdateDocument", mock.Anything, "test-ambulance", mock.Anything)

}
