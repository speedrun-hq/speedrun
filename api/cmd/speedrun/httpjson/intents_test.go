package httpjson

import (
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/speedrun-hq/speedrun/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIntents(t *testing.T) {
	const (
		validID        = "0x1234567890123456789012345678901234567890123456789012345678901234"
		validRecipient = "0x1234567890123456789012345678901234567890"
		validSender    = "0x0987654321098765432109876543210987654321"
	)

	t.Run("Create", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name           string
			request        any
			expectedStatus int
			setup          func(ts *testSuite)
		}{
			{
				name: "ValidCreation",
				request: models.CreateIntentRequest{
					ID:               validID,
					SourceChain:      1,
					DestinationChain: 2,
					Token:            "ETH",
					Amount:           "1.0",
					Recipient:        validRecipient,
					Sender:           validSender,
					IntentFee:        "0.1",
				},
				setup: func(ts *testSuite) {
					out := &models.Intent{
						ID:               validID,
						SourceChain:      1,
						DestinationChain: 2,
						Token:            "ETH",
						Amount:           "1.0",
						Recipient:        validRecipient,
						Sender:           validSender,
					}

					ts.IntentServices[1].
						On("CreateIntent", numOfArgs(9)...).
						Return(out, nil)
				},
				expectedStatus: http.StatusCreated,
			},
			{
				name:           "InvalidRequest",
				request:        "invalid json",
				expectedStatus: http.StatusBadRequest,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// ARRANGE
				ts := newTestSuite(t)

				if tt.setup != nil {
					tt.setup(ts)
				}

				// ACT
				res, err := ts.Client.Post().AddPath("/api/v1/intents").JSON(tt.request).Do()

				// ASSERT
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.StatusCode)
			})
		}
	})

	t.Run("Get", func(t *testing.T) {
		t.Parallel()

		mockIntent := &models.Intent{
			ID:               validID,
			SourceChain:      1,
			DestinationChain: 2,
			Token:            "ETH",
			Amount:           "1.0",
			Recipient:        validRecipient,
			Sender:           validSender,
			IntentFee:        "0.1",
			Status:           models.IntentStatusPending,
		}

		tests := []struct {
			name           string
			intentID       string
			expectedStatus int
			setup          func(ts *testSuite)
		}{
			{
				name:           "ValidIntentRetrieval",
				intentID:       validID,
				expectedStatus: http.StatusOK,
				setup: func(ts *testSuite) {
					ts.IntentServices[1].On("GetIntent", mock.Anything, validID).Return(mockIntent, nil)
				},
			},
			{
				name:           "IntentNotFound",
				intentID:       "0x123456789012345678901fff5678901234567890123456789012345678901230",
				expectedStatus: http.StatusNotFound,
				setup: func(ts *testSuite) {
					ts.IntentServices[1].
						On("GetIntent", mock.Anything, mock.Anything).
						Return(nil, errors.New("not found"))
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// ARRANGE
				ts := newTestSuite(t)

				if tt.setup != nil {
					tt.setup(ts)
				}

				// ACT
				res, err := ts.Client.Get().
					AddPath("/api/v1/intents/:id").
					Param("id", tt.intentID).
					Do()

				// ASSERT
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.StatusCode, res.String())

				if tt.expectedStatus == http.StatusOK {
					assertResponseContainsJSON(t, res, "id", mockIntent.ID)
					assertResponseContainsJSON(t, res, "token", mockIntent.Token)
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		t.Parallel()

		const (
			validID1       = "0x1234567890123456789012345678901234567890123456789012345678901234"
			validID2       = "0x5678901234567890123456789012345678901234567890123456789012345678"
			validRecipient = "0x1234567890123456789012345678901234567890"
			validSender    = "0x0987654321098765432109876543210987654321"
		)

		mockIntents := []*models.Intent{
			{
				ID:               validID1,
				SourceChain:      1,
				DestinationChain: 2,
				Token:            "ETH",
				Amount:           "1.0",
				Recipient:        validRecipient,
				Sender:           validSender,
				IntentFee:        "0.1",
				Status:           models.IntentStatusPending,
			},
			{
				ID:               validID2,
				SourceChain:      1,
				DestinationChain: 2,
				Token:            "ETH",
				Amount:           "2.0",
				Recipient:        validRecipient,
				Sender:           validSender,
				IntentFee:        "0.2",
				Status:           models.IntentStatusPending,
			},
		}

		tests := []struct {
			name           string
			expectedStatus int
			setup          func(ts *testSuite)
		}{
			{
				name:           "SuccessfulList",
				expectedStatus: http.StatusOK,
				setup: func(ts *testSuite) {
					ts.Database.On("ListIntentsPaginatedOptimized", numOfArgs(4)...).Return(mockIntents, 2, nil)
				},
			},
			{
				name:           "DatabaseError",
				expectedStatus: http.StatusInternalServerError,
				setup: func(ts *testSuite) {
					ts.Database.On("ListIntentsPaginatedOptimized", numOfArgs(4)...).Return(nil, 0, assert.AnError)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// ARRANGE
				ts := newTestSuite(t)

				if tt.setup != nil {
					tt.setup(ts)
				}

				// ACT
				res, err := ts.Client.Get().AddPath("/api/v1/intents").Do()

				// ASSERT
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.StatusCode)
			})
		}
	})

	t.Run("GetBySender", func(t *testing.T) {
		t.Parallel()

		const (
			validID1       = "0x1234567890123456789012345678901234567890123456789012345678901234"
			validID2       = "0x5678901234567890123456789012345678901234567890123456789012345678"
			validRecipient = "0x1234567890123456789012345678901234567890"
			validSender    = "0x0987654321098765432109876543210987654321"
			invalidSender  = "invalid-address"
		)

		mockIntents := []*models.Intent{
			{
				ID:               validID1,
				SourceChain:      1,
				DestinationChain: 2,
				Token:            "ETH",
				Amount:           "1.0",
				Recipient:        validRecipient,
				Sender:           validSender,
				IntentFee:        "0.1",
				Status:           models.IntentStatusPending,
			},
			{
				ID:               validID2,
				SourceChain:      1,
				DestinationChain: 2,
				Token:            "ETH",
				Amount:           "2.0",
				Recipient:        validRecipient,
				Sender:           validSender,
				IntentFee:        "0.2",
				Status:           models.IntentStatusPending,
			},
		}

		tests := []struct {
			name           string
			senderAddress  string
			expectedStatus int
			setup          func(ts *testSuite)
		}{
			{
				name:           "ValidSenderAddress",
				senderAddress:  validSender,
				expectedStatus: http.StatusOK,
				setup: func(ts *testSuite) {
					ts.Database.
						On("ListIntentsBySenderPaginatedOptimized", numOfArgs(4)...).
						Return(mockIntents, 2, nil)
				},
			},
			{
				name:           "InvalidSenderAddress",
				senderAddress:  invalidSender,
				expectedStatus: http.StatusBadRequest,
				setup: func(ts *testSuite) {
					// No database calls expected - validation will fail first
				},
			},
			{
				name:           "DatabaseError",
				senderAddress:  validSender,
				expectedStatus: http.StatusInternalServerError,
				setup: func(ts *testSuite) {
					ts.Database.
						On("ListIntentsBySenderPaginatedOptimized", numOfArgs(4)...).
						Return([]*models.Intent{}, 0, assert.AnError)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// ARRANGE
				ts := newTestSuite(t)

				if tt.setup != nil {
					tt.setup(ts)
				}

				// ACT
				res, err := ts.Client.Get().
					AddPath("/api/v1/intents/sender/:sender").
					Param("sender", tt.senderAddress).
					Do()

				// ASSERT
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.StatusCode)
			})
		}
	})

	t.Run("GetByRecipient", func(t *testing.T) {
		t.Parallel()

		const (
			validID1         = "0x1234567890123456789012345678901234567890123456789012345678901234"
			validID2         = "0x5678901234567890123456789012345678901234567890123456789012345678"
			validRecipient   = "0x1234567890123456789012345678901234567890"
			validSender      = "0x0987654321098765432109876543210987654321"
			invalidRecipient = "invalid-address"
		)

		mockIntents := []*models.Intent{
			{
				ID:               validID1,
				SourceChain:      1,
				DestinationChain: 2,
				Token:            "ETH",
				Amount:           "1.0",
				Recipient:        validRecipient,
				Sender:           validSender,
				IntentFee:        "0.1",
				Status:           models.IntentStatusPending,
			},
			{
				ID:               validID2,
				SourceChain:      1,
				DestinationChain: 2,
				Token:            "ETH",
				Amount:           "2.0",
				Recipient:        validRecipient,
				Sender:           validSender,
				IntentFee:        "0.2",
				Status:           models.IntentStatusPending,
			},
		}

		tests := []struct {
			name             string
			recipientAddress string
			queryParams      map[string]string
			expectedStatus   int
			setup            func(ts *testSuite)
		}{
			{
				name:             "ValidRecipientAddress",
				recipientAddress: validRecipient,
				queryParams:      map[string]string{},
				expectedStatus:   http.StatusOK,
				setup: func(ts *testSuite) {
					ts.Database.
						On("ListIntentsByRecipientPaginatedOptimized", numOfArgs(4)...).
						Return(mockIntents, 2, nil)
				},
			},
			{
				name:             "ValidRecipientAddressWithPagination",
				recipientAddress: validRecipient,
				queryParams:      map[string]string{"page": "2", "page_size": "10"},
				expectedStatus:   http.StatusOK,
				setup: func(ts *testSuite) {
					ts.Database.
						On("ListIntentsByRecipientPaginatedOptimized", numOfArgs(4)...).
						Return([]*models.Intent{}, 0, nil)
				},
			},
			{
				name:             "InvalidRecipientAddress",
				recipientAddress: invalidRecipient,
				queryParams:      map[string]string{},
				expectedStatus:   http.StatusBadRequest,
				setup: func(ts *testSuite) {
					// No database calls expected - validation will fail first
				},
			},
			{
				name:             "InvalidPageParameter",
				recipientAddress: validRecipient,
				queryParams:      map[string]string{"page": "invalid"},
				expectedStatus:   http.StatusBadRequest,
				setup: func(ts *testSuite) {
					// No database calls expected - validation will fail first
				},
			},
			{
				name:             "InvalidPageSizeParameter",
				recipientAddress: validRecipient,
				queryParams:      map[string]string{"page_size": "invalid"},
				expectedStatus:   http.StatusBadRequest,
				setup: func(ts *testSuite) {
					// No database calls expected - validation will fail first
				},
			},
			{
				name:             "PageSizeTooLarge",
				recipientAddress: validRecipient,
				queryParams:      map[string]string{"page_size": "101"},
				expectedStatus:   http.StatusBadRequest,
				setup: func(ts *testSuite) {
					// No database calls expected - validation will fail first
				},
			},
			{
				name:             "DatabaseError",
				recipientAddress: validRecipient,
				queryParams:      map[string]string{},
				expectedStatus:   http.StatusInternalServerError,
				setup: func(ts *testSuite) {
					ts.Database.
						On("ListIntentsByRecipientPaginatedOptimized", mock.Anything, validRecipient, 1, 20).
						Return([]*models.Intent{}, 0, assert.AnError)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				// ARRANGE
				ts := newTestSuite(t)

				if tt.setup != nil {
					tt.setup(ts)
				}

				// ACT
				res, err := ts.Client.Get().
					AddPath("/api/v1/intents/recipient/:recipient").
					Param("recipient", tt.recipientAddress).
					SetQueryParams(tt.queryParams).
					Do()

				// ASSERT
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, res.StatusCode)
			})
		}
	})
}
