package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSvelteService(t *testing.T) {
	svc := NewSvelteService()
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.components)
}

func TestSvelteService_Create(t *testing.T) {
	ctx := context.Background()
	svc := NewSvelteService()

	tests := []struct {
		name      string
		inputName string
		component string
		props     map[string]interface{}
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Success",
			inputName: "TestComponent",
			component: "Button",
			props:     map[string]interface{}{"variant": "primary"},
			wantErr:   false,
		},
		{
			name:      "Success without props",
			inputName: "SimpleComponent",
			component: "Div",
			props:     nil,
			wantErr:   false,
		},
		{
			name:      "Empty name",
			inputName: "",
			component: "Button",
			props:     nil,
			wantErr:   true,
			errMsg:    "name cannot be empty",
		},
		{
			name:      "Empty component",
			inputName: "Test",
			component: "",
			props:     nil,
			wantErr:   true,
			errMsg:    "component cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Create(ctx, tt.inputName, tt.component, tt.props)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEqual(t, uuid.Nil, result.ID)
				assert.Equal(t, tt.inputName, result.Name)
				assert.Equal(t, tt.component, result.Component)
			}
		})
	}
}

func TestSvelteService_Get(t *testing.T) {
	ctx := context.Background()
	svc := NewSvelteService()

	// Create a component first
	created, err := svc.Create(ctx, "TestComponent", "Button", nil)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Success",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "Not found",
			id:      uuid.New(),
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name:    "Invalid UUID",
			id:      uuid.Nil,
			wantErr: true,
			errMsg:  "invalid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Get(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, created.ID, result.ID)
			}
		})
	}
}

func TestSvelteService_Update(t *testing.T) {
	ctx := context.Background()
	svc := NewSvelteService()

	// Create a component first
	created, err := svc.Create(ctx, "Original", "Button", map[string]interface{}{"key": "value"})
	require.NoError(t, err)

	tests := []struct {
		name      string
		id        uuid.UUID
		newName   string
		component string
		props     map[string]interface{}
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "Update name",
			id:        created.ID,
			newName:   "Updated",
			component: "",
			props:     nil,
			wantErr:   false,
		},
		{
			name:      "Not found",
			id:        uuid.New(),
			newName:   "Test",
			component: "",
			props:     nil,
			wantErr:   true,
			errMsg:    "not found",
		},
		{
			name:      "Invalid UUID",
			id:        uuid.Nil,
			newName:   "Test",
			component: "",
			props:     nil,
			wantErr:   true,
			errMsg:    "invalid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.Update(ctx, tt.id, tt.newName, tt.component, tt.props)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}

	// Verify the update persisted
	updated, err := svc.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
}

func TestSvelteService_Delete(t *testing.T) {
	ctx := context.Background()
	svc := NewSvelteService()

	// Create a component first
	created, err := svc.Create(ctx, "ToDelete", "Button", nil)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Success",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "Not found (already deleted)",
			id:      created.ID,
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name:    "Invalid UUID",
			id:      uuid.Nil,
			wantErr: true,
			errMsg:  "invalid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Delete(ctx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSvelteService_List(t *testing.T) {
	ctx := context.Background()
	svc := NewSvelteService()

	// Initially empty
	list, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)

	// Add some components
	_, err = svc.Create(ctx, "Component1", "Button", nil)
	require.NoError(t, err)
	_, err = svc.Create(ctx, "Component2", "Input", nil)
	require.NoError(t, err)

	// Should have 2 components
	list, err = svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}
