package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	workflowManagerV2 "github.com/OpenNSW/go-temporal-workflow"
	"github.com/OpenNSW/nsw/internal/profile/cha"
	"github.com/OpenNSW/nsw/internal/workflow/model"
)

// MockTemplateProvider implements TemplateProvider for testing.
type MockTemplateProvider struct {
	mock.Mock
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlow(ctx context.Context, hsCodeID string, flow model.ConsignmentFlow) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, hsCodeID, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByHSCodeIDAndFlowV2(ctx context.Context, hsCodeID string, flow model.ConsignmentFlow) (*model.WorkflowTemplateV2, error) {
	args := m.Called(ctx, hsCodeID, flow)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplateV2), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByID(ctx context.Context, id string) (*model.WorkflowTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowTemplateByIDV2(ctx context.Context, id string) (*model.WorkflowTemplateV2, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowTemplateV2), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowNodeTemplatesByIDs(ctx context.Context, ids []string) ([]model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.WorkflowNodeTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetWorkflowNodeTemplateByID(ctx context.Context, id string) (*model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNodeTemplate), args.Error(1)
}

func (m *MockTemplateProvider) GetEndNodeTemplate(ctx context.Context) (*model.WorkflowNodeTemplate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.WorkflowNodeTemplate), args.Error(1)
}

// MockWMV2 implements workflowManagerV2.TemporalManager for testing.
type MockWMV2 struct {
	mock.Mock
}

func (m *MockWMV2) StartWorkflow(ctx context.Context, ID string, workflowDefinition workflowManagerV2.WorkflowDefinition, initialWorkflowVariables map[string]any) error {
	args := m.Called(ctx, ID, workflowDefinition, initialWorkflowVariables)
	return args.Error(0)
}

func (m *MockWMV2) TaskDone(ctx context.Context, workflowID, runID, nodeID string, output map[string]any) error {
	args := m.Called(ctx, workflowID, runID, nodeID, output)
	return args.Error(0)
}

func (m *MockWMV2) TaskUpdate(ctx context.Context, workflowID, runID string, update workflowManagerV2.UpdateEvent) error {
	args := m.Called(ctx, workflowID, runID, update)
	return args.Error(0)
}

func (m *MockWMV2) GetStatus(ctx context.Context, workflowID string) (*workflowManagerV2.WorkflowInstance, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*workflowManagerV2.WorkflowInstance), args.Error(1)
}

// MockCHAService implements cha.Service for testing.
type MockCHAService struct {
	mock.Mock
}

func (m *MockCHAService) GetByID(ctx context.Context, id string) (*cha.Record, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cha.Record), args.Error(1)
}

func (m *MockCHAService) GetByEmail(ctx context.Context, email string) (*cha.Record, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cha.Record), args.Error(1)
}

func (m *MockCHAService) List(ctx context.Context) ([]cha.Record, error) {
	args := m.Called(ctx)
	return args.Get(0).([]cha.Record), args.Error(1)
}

func (m *MockCHAService) Health() error {
	return m.Called().Error(0)
}

// --- CreateConsignmentShell ---

func TestConsignmentService_CreateConsignmentShell_CHANotFound(t *testing.T) {
	db, _ := setupTestDB(t)
	mockCHA := new(MockCHAService)
	svc := NewConsignmentService(db, nil, mockCHA)
	ctx := context.Background()
	chaID := uuid.NewString()

	mockCHA.On("GetByID", ctx, chaID).Return(nil, cha.ErrCHANotFound)

	result, err := svc.CreateConsignmentShell(ctx, model.ConsignmentFlowImport, chaID, "trader1")
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, cha.ErrCHANotFound))
	mockCHA.AssertExpectations(t)
}

func TestConsignmentService_CreateConsignmentShell_Success(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockCHA := new(MockCHAService)
	svc := NewConsignmentService(db, nil, mockCHA)
	ctx := context.Background()
	chaID := uuid.NewString()
	consignmentID := uuid.NewString()
	traderID := "trader1"

	mockCHA.On("GetByID", ctx, chaID).Return(&cha.Record{ID: chaID, Name: "Test CHA"}, nil)

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(`INSERT INTO "consignments"`).WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "cha_id", "state", "items"}).
			AddRow(consignmentID, "IMPORT", traderID, chaID, "INITIALIZED", []byte("[]")))

	result, err := svc.CreateConsignmentShell(ctx, model.ConsignmentFlowImport, chaID, traderID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, consignmentID, result.ID)
	mockCHA.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentByID(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	mockWM := new(MockWMV2)
	svc := NewConsignmentService(db, nil, nil)
	require.NoError(t, svc.RegisterWorkflowManager(mockWM))

	ctx := context.Background()
	consignmentID := uuid.NewString()
	hsCodeID := uuid.NewString()

	sqlMock.ExpectQuery(`SELECT \* FROM "consignments" WHERE id = \$1 ORDER BY "consignments"."id" LIMIT \$2`).
		WithArgs(consignmentID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "flow", "trader_id", "state", "created_at", "updated_at", "items"}).
			AddRow(consignmentID, "IMPORT", "trader1", "IN_PROGRESS", time.Now(), time.Now(), []byte(`[{"hsCodeId":"`+hsCodeID+`"}]`)))

	mockWM.On("GetStatus", ctx, consignmentID).Return((*workflowManagerV2.WorkflowInstance)(nil), nil)

	sqlMock.ExpectQuery(`SELECT \* FROM "hs_codes" WHERE id IN`).
		WithArgs(hsCodeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "hs_code", "description", "category"}).
			AddRow(hsCodeID, "1234.56", "Test Description", "Test Category"))

	result, err := svc.GetConsignmentByID(ctx, consignmentID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, consignmentID, result.ID)
	assert.Len(t, result.WorkflowNodes, 0)
	mockWM.AssertExpectations(t)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentsByTraderID_Empty(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewConsignmentService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments" WHERE trader_id = \$1`).
		WithArgs(traderID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	limit := 10
	offset := 0
	result, err := svc.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, model.ConsignmentFilter{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.TotalCount)
	assert.Empty(t, result.Items)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestConsignmentService_GetConsignmentsByTraderID_CountError(t *testing.T) {
	db, sqlMock := setupTestDB(t)
	svc := NewConsignmentService(db, nil, nil)
	ctx := context.Background()
	traderID := "trader1"

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "consignments"`).
		WillReturnError(errors.New("count error"))

	limit := 10
	offset := 0
	result, err := svc.GetConsignmentsByTraderID(ctx, traderID, &offset, &limit, model.ConsignmentFilter{})
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}
