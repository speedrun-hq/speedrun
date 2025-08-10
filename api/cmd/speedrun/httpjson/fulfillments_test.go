package httpjson

import (
	"net/http"
	"testing"

	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFulfillments(t *testing.T) {
	t.Skip()

	t.Run("Create", func(t *testing.T) {
		const (
			validID       = "0x1234567890123456789012345678901234567890123456789012345678901234"
			validTxHash   = "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
			validAsset    = "ETH"
			validAmount   = "1.0"
			validReceiver = "0x1234567890123456789012345678901234567890"
		)

		tests := []struct {
			name           string
			requestBody    any
			expectedStatus int
			setup          func(ts *testSuite)
		}{
			{
				name: "Valid Fulfillment Creation",
				requestBody: models.CreateFulfillmentRequest{
					IntentID: validID,
					TxHash:   validTxHash,
				},
				expectedStatus: http.StatusCreated,
				setup: func(ts *testSuite) {
					ts.FulfillmentServices[1].
						On("CreateFulfillment", mock.Anything, validID, validTxHash).
						Return(nil)

					matcher := mock.MatchedBy(func(f *models.Fulfillment) bool {
						return f.ID == validID && f.TxHash == validTxHash
					})

					ts.Database.On("CreateFulfillment", mock.Anything, matcher).Return(nil)
				},
			},
			{
				name:           "Invalid Request Body",
				requestBody:    "invalid json",
				expectedStatus: http.StatusBadRequest,
			},
			{
				name: "Invalid Chain ID",
				requestBody: models.CreateFulfillmentRequest{
					IntentID: validID,
					TxHash:   validTxHash,
				},
				expectedStatus: http.StatusBadRequest,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// ARRANGE
				ts := newTestSuite(t)

				if tt.setup != nil {
					tt.setup(ts)
				}

				// ACT
				resp, err := ts.Client.Post().AddPath("/api/v1/fulfillments").JSON(tt.requestBody).Do()

				// ASSERT
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			})
		}
	})

	t.Run("Get", func(t *testing.T) {
		const (
			validID = "0x1234567890123456789012345678901234567890123456789012345678901234"
		)

		mockFulfillment := &models.Fulfillment{
			ID:     validID,
			TxHash: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		}

		tests := []struct {
			name           string
			fulfillmentID  string
			expectedStatus int
			setup          func(ts *testSuite)
		}{
			{
				name:           "Valid Fulfillment Retrieval",
				fulfillmentID:  validID,
				expectedStatus: http.StatusOK,
				setup: func(ts *testSuite) {
					ts.Database.On("GetFulfillment", mock.Anything, validID).Return(mockFulfillment, nil)
				},
			},
			{
				name:           "Invalid Fulfillment ID Format",
				fulfillmentID:  "invalid-id",
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "Fulfillment Not Found",
				fulfillmentID:  validID,
				expectedStatus: http.StatusNotFound,
				setup: func(ts *testSuite) {
					ts.Database.On("GetFulfillment", mock.Anything, validID).Return(nil, assert.AnError)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// ARRANGE
				ts := newTestSuite(t)

				if tt.setup != nil {
					tt.setup(ts)
				}

				// ACT
				resp, err := ts.Client.Get().
					AddPath("/api/v1/fulfillments/:id").
					Param("id", tt.fulfillmentID).
					Do()

				// ASSERT
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)

				if tt.expectedStatus == http.StatusOK {
					assertResponseContainsJSON(t, resp, "fulfillment.id", mockFulfillment.ID)
					assertResponseContainsJSON(t, resp, "fulfillment.tx_hash", mockFulfillment.TxHash)
				}
			})
		}
	})

}
