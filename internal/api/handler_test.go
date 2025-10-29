package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "crypto-tracker-trader/internal/model"
    "crypto-tracker-trader/internal/service"
    "crypto-tracker-trader/internal/store"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestGetPortfolioHistory(t *testing.T) {
    // Set up
    gin.SetMode(gin.TestMode)
    r := gin.Default()

    mockStore := new(store.MockPortfolioStore)
    portfolioService := service.NewPortfolioService(mockStore)
    apiHandler := NewAPI(portfolioService)

    apiHandler.RegisterRoutes(r)

    // Mock data
    snapshot := model.PortfolioSnapshot{
        Timestamp: time.Now(),
        Assets: []model.PortfolioAsset{
            {AssetID: "BTC", Quantity: 1, Value: 50000},
        },
        TotalValue: 50000,
    }
    snapshots := []model.PortfolioSnapshot{snapshot}

    mockStore.On("GetHistory").Return(snapshots, nil)

    // Make request
    req, _ := http.NewRequest(http.MethodGet, "/api/v1/portfolio/history", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, http.StatusOK, w.Code)

    var response []model.PortfolioSnapshot
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)

    assert.Len(t, response, 1)
    assert.Equal(t, snapshot.TotalValue, response[0].TotalValue)
    assert.Len(t, response[0].Assets, 1)
    assert.Equal(t, snapshot.Assets[0].AssetID, response[0].Assets[0].AssetID)

    mockStore.AssertExpectations(t)
}
